# bridge-relay.py PORT PID serves PORT on the host loopback and relays each two-line bridge command into the
# network namespace of process PID, printing each exchange (scripts/regress-bepinex.sh, mx_bridge_relay).
import socket, subprocess, sys
port, pid = int(sys.argv[1]), sys.argv[2]
one = ("import socket,sys\ns=socket.create_connection(('127.0.0.1',%d),timeout=10)\n"
       "s.sendall(sys.stdin.buffer.read())\nsys.stdout.buffer.write(s.makefile('rb').readline())") % port
srv = socket.socket()
srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
srv.bind(("127.0.0.1", port))
srv.listen()
while True:
    c, _ = srv.accept()
    with c:
        try:
            f = c.makefile("rb")
            req = f.readline() + f.readline()
            out = subprocess.run(["nsenter", "--target", pid, "--user", "--net", "--preserve-credentials", "python3", "-c", one],
                                 input=req, capture_output=True, timeout=15)
            c.sendall(out.stdout)
            command = req.splitlines()[-1:]
            print(f"{command!r} -> {out.stdout[:80]!r} {out.stderr[-200:]!r}", flush=True)
        except Exception as e:
            print(f"relay: {e}", flush=True)
