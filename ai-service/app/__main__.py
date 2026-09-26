import logging
import sys
from pathlib import Path

import uvicorn
from dotenv import load_dotenv

from app.config import ConfigError, load_settings
from app.main import create_app

REPOSITORY_ENV_FILE = Path(__file__).resolve().parents[2] / ".env"


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
    load_dotenv(REPOSITORY_ENV_FILE)

    try:
        settings = load_settings()
    except ConfigError as error:
        logging.getLogger("app").error("invalid configuration: %s", error)
        sys.exit(1)

    uvicorn.run(create_app(settings), host="0.0.0.0", port=settings.port)


if __name__ == "__main__":
    main()
