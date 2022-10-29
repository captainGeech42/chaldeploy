#!/usr/bin/env python3

# mock the rCTF API routes necessary for chaldeploy to support offline dev

import logging
from flask import Flask, request

app = Flask(__name__)

BEARER_TOKEN = "defavalidtokenasdflkasdlfjdsalkjflkdsajfldkafkldjkls"
TEAM_ID = "b11fc3d2-ed33-4955-8ccf-01c84620b883"
TEAM_NAME = "g33chpwn"

@app.post("/api/v1/auth/login")
def login():
    app.logger.info("handling rCTF login, request data: %s", request.data.decode())

    return {"kind": "goodLogin", "message": "hello world", "data": {"authToken": BEARER_TOKEN}}

@app.get("/api/v1/users/me")
def userinfo():
    bearerStatus = "!UNSET!"
    s = request.headers["Authorization"].split(" ")
    if len(s) == 2:
        bearerStatus = "valid" if s[1] == BEARER_TOKEN else "!INVALID!"

    app.logger.info("handling rCTF /users/me, bearer token is %s", bearerStatus)

    if bearerStatus != "valid":
        return {"kind": "badUserData", "message": "invalid auth"}, 403

    return {"kind": "goodUserData", "message": "hello world", "data": {"name": TEAM_NAME, "id": TEAM_ID}}

app.logger.setLevel(logging.INFO)
app.run(host="localhost", port="6666")