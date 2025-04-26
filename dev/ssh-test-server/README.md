# SSH Test Server

This is a dev-only Ubuntu SSH server for testing mo-sh server flows locally.
It runs a real `sshd` as `root` with public-key auth on host port `2222`.

The keypair in `keys/` is intentionally committed because it is only for this
local fixture. Do not use it anywhere outside this test container.

Start it:

```sh
docker compose up -d ssh-test-server
```

Connect manually:

```sh
ssh -i dev/ssh-test-server/keys/id_ed25519 -p 2222 root@localhost
```

Use these values when creating a server in the app:

- hostname: `localhost`
- port: `2222`
- username used by app: `root`
- private key: contents of `dev/ssh-test-server/keys/id_ed25519`

The app uses strict known-hosts checking, so add the container host key before testing:

```sh
ssh-keyscan -p 2222 localhost >> ~/.ssh/known_hosts
```
