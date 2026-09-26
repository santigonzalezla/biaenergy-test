import logging
import sys

import uvicorn

from app.config import ConfigError, load_settings
from app.main import create_app


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")

    try:
        settings = load_settings()
    except ConfigError as error:
        logging.getLogger("app").error("invalid configuration: %s", error)
        sys.exit(1)

    uvicorn.run(create_app(settings), host="0.0.0.0", port=settings.port)


if __name__ == "__main__":
    main()
