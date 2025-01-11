FROM python:3.8-openbsd.10

ADD . /code
WORKDIR /code

ADD requirements /requirements
RUN pip install -r /requirements/test.txt
