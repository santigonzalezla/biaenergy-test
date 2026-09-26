import logging
from datetime import UTC, datetime
from typing import Any

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException

logger = logging.getLogger(__name__)

HTTP_ERROR_CODES = {
    404: "ROUTE_NOT_FOUND",
    405: "METHOD_NOT_ALLOWED",
}


def register_error_handlers(app: FastAPI) -> None:
    app.add_exception_handler(RequestValidationError, _validation_error)
    app.add_exception_handler(HTTPException, _http_error)
    app.add_exception_handler(Exception, _unexpected_error)


def error_response(request: Request, status: int, code: str, message: str, details: Any = None) -> JSONResponse:
    error: dict[str, Any] = {
        "code": code,
        "message": message,
        "path": request.url.path,
        "timestamp": datetime.now(UTC).isoformat(),
    }
    if details is not None:
        error["details"] = details
    return JSONResponse(status_code=status, content={"error": error})


async def _validation_error(request: Request, exc: RequestValidationError) -> JSONResponse:
    details = [
        {"field": ".".join(str(part) for part in error["loc"][1:]), "message": error["msg"]} for error in exc.errors()
    ]
    return error_response(request, 400, "VALIDATION_ERROR", "The request contains invalid data", details)


async def _http_error(request: Request, exc: HTTPException) -> JSONResponse:
    code = HTTP_ERROR_CODES.get(exc.status_code, "HTTP_ERROR")
    return error_response(request, exc.status_code, code, str(exc.detail))


async def _unexpected_error(request: Request, exc: Exception) -> JSONResponse:
    logger.exception("unexpected error", extra={"path": request.url.path})
    return error_response(request, 500, "INTERNAL_ERROR", "Internal server error")
