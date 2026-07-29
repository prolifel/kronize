FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    gnupg \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir \
    google-api-python-client \
    google-auth-httplib2 \
    google-auth-oauthlib \
    boto3 \
    botocore \
    requests \
    pyyaml \
    python-dotenv \
    beautifulsoup4

WORKDIR /code
ENTRYPOINT ["python"]
