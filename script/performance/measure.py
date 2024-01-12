from __modules__ import *
from sys import argv
from time import sleep
from pprint import pprint

LOGFILE = "/tmp/urunc.zlog"
DELAY = 2


def main():
    if len(argv) != 2:
        print("Error: Iterations not specified!")
        print("")
        print("Usage:")
        print(f"\t{argv[0]} <ITERATIONS>")
        exit(1)
    iterations = int(argv[1])
    myprint(f"Collecting timestamps for {iterations} iterations")
    sleep(2)
    emptyFile(filename=LOGFILE)
    containerIDs = []
    for i in range(iterations):
        myprint(f"Running iteration {i+1} of {iterations}")
        containerID = spawnContainer()
        containerIDs.append(containerID)
        sleep(DELAY)
        success = deleteContainer()
        if not success:
            print("Error removing container.")
            exit(1)
    myprint("Done")
    timestampDiffs = {}
    for containerID in containerIDs:
        data = parseSingleContainerTimestamps(
            filename=LOGFILE, containerID=containerID)
        series = TimestampSeries(data=data)
        diffs = series.diffs
        for key in diffs:
            value = diffs[key]
            if key in timestampDiffs:
                timestampDiffs[key].append(value)
            else:
                timestampDiffs[key] = [value]
    result = {}
    for key in timestampDiffs:
        value = timestampDiffs[key]
        current = timestampDiffs[key]
        durations = [c.duration for c in current]
        max_duration = f"{max(durations)} ns"
        min_duration = f"{min(durations)} ns"
        avg_duration = f"{int(sum(durations)/len(durations))} ns"
        result[key] = {"maximum": max_duration,
                       "minimum": min_duration, "average": avg_duration}
    pprint(result)
    # emptyFile(filename=LOGFILE)


if __name__ == "__main__":
    main()