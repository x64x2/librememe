import json

import furl
import requests
import aiohttp

from . import exceptions

DEFAULT_TIMEOUT = 30


class AioClient:

    API_OK = 200
    API_ERRORS_MAPPING = {
        401: exceptions.UnauthorizedError,
        403: exceptions.OldTelemetryError,
        404: exceptions.NotFoundError,
        415: exceptions.InvalidContentTypeError,
        429: exceptions.RateLimitError,
    }

    def __init__(self):
        self.session = aiohttp.ClientSession()
        self.session.headers.update({'Accept': 'application/vnd.api+json'})
        self.url = furl.furl()
        
    async def request(self, endpoint):
        response = await self.session.get(str(endpoint), timeout=DEFAULT_TIMEOUT)

        if response.status != self.API_OK:
            exception = self.API_ERRORS_MAPPING.get(
                response.status, exceptions.APIError)
            raise exception(response_headers=response.headers)

        return await response.json()

class AioAPIClient(AioClient):

    BASE_URL = 'https://api.pubg.com/'

    def __init__(self, api_key):
        super().__init__()
        self.session.headers.update({'Authorization': 'Bearer ' + api_key})
        self.url.set(path=self.BASE_URL)


class AioTelemetryClient(AioClient):

    TELEMETRY_HOSTS = [
        'telemetry-cdn.playbattlegrounds.com'
    ]

    async def request(self, endpoint):
        if furl.furl(endpoint).host not in self.TELEMETRY_HOSTS:
            raise exceptions.TelemetryURLError
        return await super().request(endpoint)
