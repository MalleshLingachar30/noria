# Third-party notices

Noria is licensed under Apache-2.0. The native binary includes the following
third-party component, which retains its own license.

## Gorilla WebSocket v1.5.3

Source: https://github.com/gorilla/websocket

Copyright (c) 2013 The Gorilla WebSocket Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

  Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

  Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

## Separately installed components

The optional Claude quota bridge explicitly installs the pinned Anthropic
Claude Agent SDK through npm. That package is not bundled into Noria's binary
and includes its own license. Native Codex and Claude Code clients are installed
and licensed separately by their respective publishers.

Documentation font licenses are distributed beside the corresponding font
assets under `docs/public/fonts`.

## Design reference

The project reviewed CLIProxyAPI (https://github.com/router-for-me/CLIProxyAPI)
as a reference for native provider protocol integration. Noria does not vendor
CLIProxyAPI or its multi-provider execution framework.
