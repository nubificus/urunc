from __modules__ import *
from sys import argv

LOGFILE = "/tmp/urunc.zlog"


def main():
    if len(argv) != 2:
        print("Error: Container ID not specified!")
        print("")
        print("Usage:")
        print(f"\t{argv[0]} <CONTAINER_ID>")
        exit(1)
    containerID = argv[1]
    data = parseSingleContainerTimestamps(
        filename=LOGFILE, containerID=containerID)
    series = TimestampSeries(data=data)
    print(series.report)


if __name__ == "__main__":
    main()
