# Copyright (c) 2023-2024, Nubificus LTD
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
