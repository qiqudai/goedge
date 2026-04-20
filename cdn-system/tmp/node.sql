INSERT INTO node (id, pid, name, ip, token, enable, create_at, update_at)
VALUES (1, 0, 'wsl-node-1', '172.20.9.140', '1e53b4aaf19c4feda09cc353f28e4c7e', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
  pid=VALUES(pid),
  name=VALUES(name),
  ip=VALUES(ip),
  token=VALUES(token),
  enable=VALUES(enable),
  update_at=VALUES(update_at);