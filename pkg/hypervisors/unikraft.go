// TODO: move unikernel specific logic to a discrete pkg once the details are ironed out
package hypervisors

import (
	"fmt"

	//unet "github.com/nubificus/urunc/pkg/network"
)

type UnikraftCliOpts struct {
	Command string
	Net     UnikraftNet
	Blk     UnikraftBlk
}

type UnikraftNet struct {
	Address string
	Mask    string
	Gateway string
}

type UnikraftBlk struct {
	RootFs string
	DevTag string //Will be used for the 9pfs
}

func UnikraftCli(data ExecData) (string, error) {
	var cli_opts UnikraftCliOpts

	cli_opts.Command = data.CmdLine

	vmmLog.WithField("mask", data.Network.EthDevice.Mask).Error("xdss")
	cli_opts.Net.Address = "netdev.ipv4_addr=" + data.Network.EthDevice.IP
	cli_opts.Net.Gateway = "netdev.ipv4_gw_addr=" + data.Network.EthDevice.DefaultGateway
	cli_opts.Net.Mask = "netdev.ipv4_subnet_mask=" + data.Network.EthDevice.Mask
	vmmLog.WithField("net config = ", cli_opts.Net).Info("xdss")

	cli_opts.Blk.RootFs = "vfs.rootfs=" + "initrd"
	cli_opts.Blk.DevTag = ""
	vmmLog.WithField("block config = ", cli_opts.Blk).Info("xdss")

	//return fmt.Sprintf("\"%s %s %s %s %s -- %s\"", cli_opts.Net.Address,
	return fmt.Sprintf("%s %s %s %s %s -- %s", cli_opts.Net.Address,
						   cli_opts.Net.Gateway,
						   cli_opts.Net.Mask,
						   cli_opts.Blk.RootFs,
						   cli_opts.Blk.DevTag,
						   cli_opts.Command), nil
}
