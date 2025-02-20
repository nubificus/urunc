#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

crio_drop_in_conf_dir="/etc/crio/crio.conf.d/"
crio_drop_in_conf_file="${crio_drop_in_conf_dir}/99-urunc-deploy"
crio_drop_in_conf_file_debug="${crio_drop_in_conf_dir}/100-debug"

containerd_conf_file="/etc/containerd/config.toml"
containerd_conf_file_backup="${containerd_conf_file}.bak"
containerd_conf_tmpl_file=""
use_containerd_drop_in_conf_file="false"
containerd_drop_in_conf_file="/etc/containerd/config.d/urunc-deploy.toml"

IFS=' ' read -a hypervisors <<< "$HYPERVISORS"
HELM_POST_DELETE_HOOK="${HELM_POST_DELETE_HOOK:-"false"}"

function host_systemctl() {
    nsenter --target 1 --mount systemctl "${@}"
}
function print_usage() {
    echo "Usage: $0 {install|uninstall|cleanup}"
}

function install_artifact() {
    local src="$1"
    local dest="$2"
    cp "$src" "$dest"
    chmod +x "$dest"
}

function install_urunc() {
    install_artifact /urunc-artifacts/urunc /host/usr/local/bin/urunc
}


function install_shim() {
    install_artifact /urunc-artifacts/containerd-shim-urunc-v2 /host/usr/local/bin/containerd-shim-urunc-v2
}

function uninstall_artifact() {
    local artifact="$1"
    rm -f "/host/usr/local/bin/${artifact}"
}

function uninstall_urunc() {
    uninstall_artifact "urunc"
}

function uninstall_shim() {
    uninstall_artifact "containerd-shim-urunc-v2"
}

function uninstall_qemu() {
    uninstall_artifact "qemu-system-x86_64"
}

function uninstall_firecracker() {
    uninstall_artifact "firecracker"
}

function uninstall_solo5-spt() {
    uninstall_artifact "solo5-spt"
}

function uninstall_solo5-hvt() {
    uninstall_artifact "solo5-hvt"
}


function install_artifacts() {
    echo "copying urunc artifacts onto host"
    mkdir -p /host/usr/local/bin

    install_urunc
    install_shim

    # install only the hypervisors found in the HYPERVISORS environment variable
    for hypervisor in "${hypervisors[@]}" ; do
        case "$hypervisor" in
        qemu)
            echo "Installing qemu"
            install_artifact /urunc-artifacts/hypervisors/qemu-system-$(uname -m) /host/usr/local/bin/qemu-system-$(uname -m)
            mkdir -p /host/usr/local/share/qemu/
            cp /urunc-artifacts/opt/kata/share/kata-qemu/qemu/*.bin /host/usr/local/share/qemu/
            ;;
        firecracker)
            echo "Installing firecracker"
            install_artifact /urunc-artifacts/hypervisors/firecracker /host/usr/local/bin/firecracker
            ;;
        solo5-spt)
            echo "Installing solo5-spt"
            install_artifact /urunc-artifacts/hypervisors/solo5-spt /host/usr/local/bin/solo5-spt
            ;;
        solo5-hvt)
            echo "Installing solo5-hvt"
            install_artifact /urunc-artifacts/hypervisors/solo5-hvt /host/usr/local/bin/solo5-hvt
            ;;
        *)
            echo "Unsupported hypervisor: $hypervisor"
            ;;
	    esac
    done
}

function remove_artifacts() {
    uninstall_urunc
    uninstall_shim
    uninstall_qemu
    uninstall_firecracker
}

die() {
    msg="$*"
    echo "ERROR: $msg" >&2
    exit 1
}


function create_runtimeclass() {
    echo "Creating the runtime class"
    kubectl apply -f /urunc-artifacts/runtimeclasses/runtimeclass.yaml
}

function get_container_runtime() {
	local runtime=$(kubectl get node $NODE_NAME -o jsonpath='{.status.nodeInfo.containerRuntimeVersion}')
	if [ "$?" -ne 0 ]; then
                die "invalid node name"
	fi

	if echo "$runtime" | grep -qE "cri-o"; then
		echo "cri-o"
	elif echo "$runtime" | grep -qE 'containerd.*-k3s'; then
		if host_systemctl is-active --quiet rke2-agent; then
			echo "rke2-agent"
		elif host_systemctl is-active --quiet rke2-server; then
			echo "rke2-server"
		elif host_systemctl is-active --quiet k3s-agent; then
			echo "k3s-agent"
		else
			echo "k3s"
		fi
	elif host_systemctl is-active --quiet k0scontroller; then
		echo "k0s-controller"
	elif host_systemctl is-active --quiet k0sworker; then
		echo "k0s-worker"
	else
		echo "$runtime" | awk -F '[:]' '{print $1}'
	fi
}

function is_containerd_capable_of_using_drop_in_files() {
    local runtime="$1"

    if [ "$runtime" == "crio" ]; then
        # This should never happen but better be safe than sorry
        echo "false"
        return
    fi

    if [[ "$runtime" =~ ^(k0s-worker|k0s-controller)$ ]]; then
        # k0s does the work of using drop-in files better than any other "k8s distro", so
        # we don't mess up with what's being correctly done.
        echo "false"
        return
    fi

    local version_major=$(kubectl get node $NODE_NAME -o jsonpath='{.status.nodeInfo.containerRuntimeVersion}' | grep -oE '[0-9]+\.[0-9]+' | cut -d'.' -f1)
    if [ $version_major -lt 2 ]; then
        # Only containerd 2.0 does the merge of the plugins section from different snippets,
        # instead of overwritting the whole section, which makes things considerably more
        # complicated for us to deal with.
        #
        # It's been discussed with containerd community, and the patch needed will **NOT** be
        # backported to the release 1.7, as that breaks the behaviour from an existing release.
        echo "false"
        return
    fi

    echo "true"
}


function wait_till_node_is_ready() {
    local ready="False"

    while ! [[ "${ready}" == "True" ]]; do
        sleep 2s
        ready=$(kubectl get node $NODE_NAME -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')
    done
}

function configure_crio() {
    # Configure crio to use urunc:
    echo "Add urunc as a supported runtime for CRIO:"

    echo "Drop-in configuration directory: $crio_drop_in_conf_dir"
    echo "Drop-in configuration file: $crio_drop_in_conf_file"
    echo "Drop-in debug file: $crio_drop_in_conf_file_debug"


    # As we don't touch the original configuration file in any way,
    # let's just ensure we remove any exist configuration from a
    # previous deployment.
    mkdir -p "$crio_drop_in_conf_dir"
    rm -f "$crio_drop_in_conf_file"
    touch "$crio_drop_in_conf_file"
    rm -f "$crio_drop_in_conf_file_debug"
    touch "$crio_drop_in_conf_file_debug"

    local urunc_path="/usr/local/bin/containerd-shim-urunc-v2"
    local urunc_conf="crio.runtime.runtimes.urunc"

	cat <<EOF | tee -a "$crio_drop_in_conf_file"

[$urunc_conf]
	runtime_path = "${urunc_path}"
	runtime_type = "vm"
	runtime_root = "/run/urunc"
	privileged_without_host_devices = true
EOF


    if [ "${DEBUG}" == "true" ]; then
		cat <<EOF | tee $crio_drop_in_conf_file_debug
[crio.runtime]
log_level = "debug"
EOF
    fi
}


function configure_cri_runtime() {
    case $1 in
        crio)
            configure_crio
            # TODO: Configure crio
            # die "crio is not supported"
        ;;
        containerd | k3s | k3s-agent | rke2-agent | rke2-server | k0s-controller | k0s-worker)
            configure_containerd "$1"
        ;;
    esac
    if [ "$1" == "k0s-worker" ] || [ "$1" == "k0s-controller" ]; then
        # do nothing, k0s will automatically load the config on the fly
        :
    else
        echo "reloading $1"
        host_systemctl daemon-reload
        host_systemctl restart "$1"
    fi

    wait_till_node_is_ready
}

function configure_containerd() {
    # Configure containerd to use urunc:
    echo "Add urunc as a supported runtime for containerd"
    echo "Containerd conf file: $containerd_conf_file"
    mkdir -p /etc/containerd/

    if [ $use_containerd_drop_in_conf_file = "false" ] && [ -f "$containerd_conf_file" ]; then
        # only backup in case drop-in files are not supported, and when doing the backup
        # only do it if a backup doesn't already exist (don't override original)
        cp -n "$containerd_conf_file" "$containerd_conf_file_backup"
    fi

    if [ $use_containerd_drop_in_conf_file = "true" ]; then
        tomlq -i -t $(printf '.imports|=.+["%s"]' ${containerd_drop_in_conf_file}) ${containerd_conf_file}
    fi
    local urunc_runtime="urunc"
    local pluginid=cri
    local configuration_file="${containerd_conf_file}"

    # Properly set the configuration file in case drop-in files are supported
    if [ $use_containerd_drop_in_conf_file = "true" ]; then
        configuration_file="/host${containerd_drop_in_conf_file}"
    fi

    local containerd_root_conf_file="$containerd_conf_file"
    if [[ "$1" =~ ^(k0s-worker|k0s-controller)$ ]]; then
        containerd_root_conf_file="/etc/containerd/containerd.toml"
    fi

    if grep -q "version = 2\>" $containerd_root_conf_file; then
        pluginid=\"io.containerd.grpc.v1.cri\"
    fi

    if grep -q "version = 3\>" $containerd_root_conf_file; then
        pluginid=\"io.containerd.cri.v1.runtime\"
    fi

    echo "Plugin ID: ${pluginid}"

    local runtime_table=".plugins.${pluginid}.containerd.runtimes.\"urunc\""
    local runtime_type=\"io.containerd.urunc.v2\"
    # local container_annotations="\[\"com.urunc.unikernel.*\"\]"
    # local pod_annotations="\[\"com.urunc.unikernel.*\"\]"
    # local snapshottter="devmapper"

    echo "Once again, configuration file is ${configuration_file}"
    # configuration_file = "/host${configuration_file}"
    mkdir -p $(dirname ${configuration_file})
    touch ${configuration_file}

    tomlq -i -t $(printf '%s.runtime_type=%s' ${runtime_table} ${runtime_type}) ${configuration_file}

    if [ "${DEBUG}" == "true" ]; then
        tomlq -i -t '.debug.level = "debug"' ${configuration_file}
    fi
}

function cleanup_cri_runtime() {
	case $1 in
	crio)
		cleanup_crio
		;;
	containerd | k3s | k3s-agent | rke2-agent | rke2-server | k0s-controller | k0s-worker)
		cleanup_containerd
		;;
	esac

	[ "${HELM_POST_DELETE_HOOK}" == "false" ] && return

	# Only run this code in the HELM_POST_DELETE_HOOK
	restart_cri_runtime "$1"
}

function cleanup_crio() {
	rm -f $crio_drop_in_conf_file
	if [[ "${DEBUG}" == "true" ]]; then
		rm -f $crio_drop_in_conf_file_debug
	fi
}

function cleanup_containerd() {
	if [ $use_containerd_drop_in_conf_file = "true" ]; then
		# There's no need to remove the drop-in file, as it'll be removed as
		# part of the artefacts removal.  Thus, simply remove the file from
		# the imports line of the containerd configuration and return.
		tomlq -i -t $(printf '.imports|=.-["%s"]' ${containerd_drop_in_conf_file}) ${containerd_conf_file}
		return
	fi

	rm -f $containerd_conf_file
	if [ -f "$containerd_conf_file_backup" ]; then
		mv "$containerd_conf_file_backup" "$containerd_conf_file"
	fi
}

function restart_cri_runtime() {
	local runtime="${1}"

	if [ "${runtime}" == "k0s-worker" ] || [ "${runtime}" == "k0s-controller" ]; then
		# do nothing, k0s will automatically unload the config on the fly
		:
	else
		host_systemctl daemon-reload
		host_systemctl restart "${runtime}"
	fi
}

function reset_runtime() {
	kubectl label node "$NODE_NAME" katacontainers.io/kata-runtime-
	restart_cri_runtime "$1"

	if [ "$1" == "crio" ] || [ "$1" == "containerd" ]; then
		host_systemctl restart kubelet
	fi

	wait_till_node_is_ready
}

function main() {
    action=${1:-}
    if [ -z "$action" ]; then
        print_usage
        die "invalid arguments"
    fi
    echo "Action:"
    echo "* $action"
    echo ""
    echo "Environment variables passed to this script"
    echo "* NODE_NAME: ${NODE_NAME}"
    echo "* HYPERVISORS: ${HYPERVISORS}"

    # verify user is root
    euid=$(id -u)
    if [[ $euid -ne 0 ]]; then
        die  "This script must be run as root"
    fi
    runtime=$(get_container_runtime)
    if [[ "$runtime" =~ ^(k3s|k3s-agent|rke2-agent|rke2-server)$ ]]; then
        containerd_conf_tmpl_file="${containerd_conf_file}.tmpl"
        containerd_conf_file_backup="${containerd_conf_tmpl_file}.bak"
    elif [[ "$runtime" =~ ^(k0s-worker|k0s-controller)$ ]]; then
        # From 1.27.1 onwards k0s enables dynamic configuration on containerd CRI runtimes.
        # This works by k0s creating a special directory in /etc/k0s/containerd.d/ where user can drop-in partial containerd configuration snippets.
        # k0s will automatically pick up these files and adds these in containerd configuration imports list.
        containerd_conf_file="/etc/containerd/containerd.d/urunc.toml"
        containerd_conf_file_backup="${containerd_conf_tmpl_file}.bak"
    fi

    use_containerd_drop_in_conf_file=$(is_containerd_capable_of_using_drop_in_files "$runtime")
    echo "Using containerd drop-in files: $use_containerd_drop_in_conf_file"

    echo "Runtime: ${runtime}"
    echo "containerd_conf_file: ${containerd_conf_file}"
    echo "containerd_conf_tmpl_file: ${containerd_conf_tmpl_file}"
    echo "containerd_conf_file_backup: ${containerd_conf_file_backup}"
    echo "Using containerd drop-in files: $use_containerd_drop_in_conf_file"

    case "$action" in
        install)
            echo "install started" >> /host/urunc-deploy.txt
            if [[ "$runtime" =~ ^(k3s|k3s-agent|rke2-agent|rke2-server)$ ]]; then
                if [ ! -f "$containerd_conf_tmpl_file" ] && [ -f "$containerd_conf_file" ]; then
                    cp "$containerd_conf_file" "$containerd_conf_tmpl_file"
                fi
                # Only set the containerd_conf_file to its new value after
                # copying the file to the template location
                containerd_conf_file="${containerd_conf_tmpl_file}"
                containerd_conf_file_backup="${containerd_conf_tmpl_file}.bak"
            elif [[ "$runtime" =~ ^(k0s-worker|k0s-controller)$ ]]; then
                mkdir -p $(dirname "$containerd_conf_file")
                touch "$containerd_conf_file"
            elif [[ "$runtime" == "containerd" ]]; then
                if [ ! -f "$containerd_conf_file" ] && [ -d $(dirname "$containerd_conf_file") ] && [ -x $(command -v containerd) ]; then
                    containerd config default > "$containerd_conf_file"
                fi
            fi
            install_artifacts
            configure_cri_runtime "$runtime"
            kubectl label node "$NODE_NAME" --overwrite urunc.io/urunc-runtime=true
            # create_runtimeclass
            echo "EVERYTHING WENT WELL"
            echo "install completed" >> /host/urunc-deploy.txt
        ;;
        cleanup)
            echo "cleanup started" >> /host/urunc-deploy.txt

            if [[ "$runtime" =~ ^(k3s|k3s-agent|rke2-agent|rke2-server)$ ]]; then
			       containerd_conf_file_backup="${containerd_conf_tmpl_file}.bak"
			       containerd_conf_file="${containerd_conf_tmpl_file}"
			fi

            local urunc_deploy_installations=$(kubectl -n kube-system get ds | grep urunc-deploy | wc -l)

			if [ "${HELM_POST_DELETE_HOOK}" == "true" ]; then
				# Remove the label as the first thing, so we ensure no more urunc
				# pods would be scheduled here.
				#
				# If we still have any other installation here, it means we'll break them
				# removing the label, so we just don't do it.
				if [ $urunc_deploy_installations -eq 0 ]; then
					kubectl label node "$NODE_NAME" urunc.io/urunc-runtime-
				fi
			fi

			cleanup_cri_runtime "$runtime"
			if [ "${HELM_POST_DELETE_HOOK}" == "false" ]; then
				# If we still have any other installation here, it means we'll break them
				# removing the label, so we just don't do it.
				if [ $urunc_deploy_installations -eq 0 ]; then
					kubectl label node "$NODE_NAME" --overwrite urunc.io/urunc-runtime=cleanup
				fi
			fi
            remove_artifacts
            echo "cleanup completed" >> /host/urunc-deploy.txt


			if [ "${HELM_POST_DELETE_HOOK}" == "true" ]; then
				# After everything was cleaned up, there's no reason to continue
				# and sleep forever.  Let's just return success..
				exit 0
			fi
			;;
		reset)
            echo "reset started" >> /host/urunc-deploy.txt
            kubectl label node "$NODE_NAME" urunc.io/urunc-runtime- # TODO: not sure if we want to remove this
			reset_runtime $runtime
            echo "reset completed" >> /host/urunc-deploy.txt
			;;
		*)
            print_usage
            die "invalid arguments"
        ;;
    esac
    # Script is being called as a Daemonset. We need to keep it running, otherwise the pod will be restarted.
    sleep infinity

}

main "$@"
