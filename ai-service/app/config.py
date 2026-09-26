import os
from dataclasses import dataclass
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError

DEFAULT_LLM_MODEL = "claude-sonnet-5"


class ConfigError(ValueError):
    pass


@dataclass(frozen=True)
class Settings:
    environment: str
    port: int
    timezone: str
    llm_api_key: str | None
    llm_model: str

    @property
    def is_production(self) -> bool:
        return self.environment == "production"

    @property
    def llm_enabled(self) -> bool:
        return self.llm_api_key is not None


def load_settings(environ: dict[str, str] | None = None) -> Settings:
    env = os.environ if environ is None else environ
    errors: list[str] = []

    raw_port = get_env(env, "PORT", "8000")
    if not raw_port.isdigit():
        errors.append(f"PORT must be a valid integer: {raw_port!r}")

    timezone = get_env(env, "APP_TIMEZONE", "America/Bogota")
    try:
        ZoneInfo(timezone)
    except (ZoneInfoNotFoundError, ValueError):
        errors.append(f"APP_TIMEZONE is not a valid IANA timezone: {timezone!r}")

    if errors:
        raise ConfigError("; ".join(errors))

    return Settings(
        environment=get_env(env, "APP_ENV", "development"),
        port=int(raw_port),
        timezone=timezone,
        llm_api_key=get_env(env, "LLM_API_KEY", "") or None,
        llm_model=get_env(env, "LLM_MODEL", DEFAULT_LLM_MODEL),
    )


def get_env(env: dict[str, str], key: str, fallback: str) -> str:
    value = env.get(key, "").strip()
    return value or fallback
