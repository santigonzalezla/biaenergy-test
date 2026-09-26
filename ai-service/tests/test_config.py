import pytest

from app.config import ConfigError, load_settings


def test_defaults_when_nothing_is_set():
    settings = load_settings({})

    assert settings.port == 8000
    assert settings.timezone == "America/Bogota"
    assert settings.llm_model == "claude-opus-5"
    assert not settings.llm_enabled
    assert not settings.is_production


def test_reads_railway_port_and_trims_values():
    settings = load_settings({"PORT": " 8080 ", "APP_ENV": "production"})

    assert settings.port == 8080
    assert settings.is_production


def test_llm_is_enabled_by_providing_a_key():
    settings = load_settings({"LLM_API_KEY": "secret", "LLM_MODEL": "another-model"})

    assert settings.llm_enabled
    assert settings.llm_model == "another-model"


def test_blank_key_keeps_the_llm_disabled():
    assert not load_settings({"LLM_API_KEY": "   "}).llm_enabled


@pytest.mark.parametrize(
    ("environ", "message"),
    [
        pytest.param({"PORT": "abc"}, "PORT must be a valid integer", id="non numeric port"),
        pytest.param({"APP_TIMEZONE": "Mars/Olympus"}, "APP_TIMEZONE", id="unknown timezone"),
    ],
)
def test_invalid_configuration(environ, message):
    with pytest.raises(ConfigError, match=message):
        load_settings(environ)


def test_reports_every_error_at_once():
    with pytest.raises(ConfigError) as error:
        load_settings({"PORT": "abc", "APP_TIMEZONE": "Mars/Olympus"})

    assert "PORT" in str(error.value)
    assert "APP_TIMEZONE" in str(error.value)
