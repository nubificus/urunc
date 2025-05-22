# SPDX-License-Identifier: Apache-2.0

from datetime import datetime


def on_config(config, **kwargs):
    author = config.get("site_author", "Author")
    year = datetime.now().year
    start_year = config.get("copyright", year)
    if start_year != year:
        config["copyright"] = f"Copyright &copy; {start_year} - {year} {author}"
    else:
        config["copyright"] = f"Copyright &copy; {year} {author}"
    return config
