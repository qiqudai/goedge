SET NAMES utf8mb4;

INSERT INTO `node_group` (`id`, `name`, `create_at`, `update_at`, `sort_order`)
VALUES (1, 'cc-e2e-group', NOW(), NOW(), 100);

INSERT INTO `node` (
  `id`, `name`, `ip`, `enable`, `level`, `create_at`, `update_at`
) VALUES (
  1, 'cc-e2e-edge', '10.89.0.2', 1, 1, NOW(3), NOW(3)
);

INSERT INTO `line` (
  `id`, `node_group_id`, `node_id`, `line_id`, `line_name`, `weight`,
  `create_at`, `update_at`, `enable`
) VALUES (
  1, 1, 1, 'default', 'default', '10', NOW(), NOW(), 1
);

INSERT INTO `cc_match` (
  `id`, `uid`, `name`, `data`, `enable`, `create_at`, `update_at`
) VALUES (
  10027, 1, 'cc-e2e-uri-root',
  '{"rules":[{"key":"uri","operator":"equals","value":"/"}]}',
  1, NOW(), NOW()
),
(
  10028, 1, 'cc-e2e-uri-api',
  '{"rules":[{"key":"uri","operator":"equals","value":"/api"}]}',
  1, NOW(), NOW()
),
(
  10029, 1, 'cc-e2e-uri-dual',
  '{"rules":[{"key":"uri","operator":"equals","value":"/dual"}]}',
  1, NOW(), NOW()
),
(
  10030, 1, 'cc-e2e-uri-admin-prefix',
  '{"rules":[{"key":"uri","operator":"prefix","value":"/admin"}]}',
  1, NOW(), NOW()
),
(
  10031, 1, 'cc-e2e-uri-secure-and-get',
  '{"rules":[{"key":"uri","operator":"eq","value":"/secure"},{"key":"method","operator":"eq","value":"GET"}]}',
  1, NOW(), NOW()
),
(
  10032, 1, 'cc-e2e-ua-badbot',
  '{"rules":[{"key":"user_agent","operator":"contains","value":"BadBot"}]}',
  1, NOW(), NOW()
),
(
  10033, 1, 'cc-e2e-uri-static-prefix',
  '{"rules":[{"key":"uri","operator":"prefix","value":"/static"}]}',
  1, NOW(), NOW()
);

INSERT INTO `cc_filter` (
  `id`, `uid`, `name`, `type`, `within_second`, `max_req`, `max_req_per_uri`,
  `extra`, `enable`, `create_at`, `update_at`
) VALUES (
  10025, 1, 'cc-e2e-rate', 'req_rate', 5, 2, 0, '{}', 1, NOW(), NOW()
),
(
  10026, 1, 'cc-e2e-dual-filter1', 'req_rate', 5, 2, 0, '{}', 1, NOW(), NOW()
),
(
  10027, 1, 'cc-e2e-dual-filter2', 'log', 5, 0, 0, '{}', 1, NOW(), NOW()
),
(
  10028, 1, 'cc-e2e-per-uri-rate', 'req_rate', 5, 0, 2, '{}', 1, NOW(), NOW()
),
(
  10029, 1, 'cc-e2e-instant-block', 'block', 5, 0, 0, '{}', 1, NOW(), NOW()
);

INSERT INTO `cc_rule` (
  `id`, `uid`, `name`, `data`, `enable`, `create_at`, `update_at`
) VALUES (
  10001, 1, 'cc-e2e-rate-rule',
  '[{"matcher":10027,"filter1":10025,"action":"block","mode":"stop","state":true}]',
  1, NOW(), NOW()
),
(
  10003, 1, 'cc-e2e-dual-filter-rule',
  '[{"matcher":10029,"filter1":10026,"filter2":10027,"action":"block","mode":"stop","state":true}]',
  1, NOW(), NOW()
),
(
  10004, 1, 'cc-e2e-admin-prefix-rate',
  '[{"matcher":10030,"filter1":10025,"action":"block","mode":"stop","state":true}]',
  1, NOW(), NOW()
),
(
  10005, 1, 'cc-e2e-static-per-uri-rate',
  '[{"matcher":10033,"filter1":10028,"action":"block","mode":"stop","state":true}]',
  1, NOW(), NOW()
),
(
  10006, 1, 'cc-e2e-secure-and-get-block',
  '[{"matcher":10031,"filter1":10029,"action":"block","mode":"stop","state":true}]',
  1, NOW(), NOW()
),
(
  10007, 1, 'cc-e2e-badbot-block',
  '[{"matcher":10032,"filter1":10029,"action":"block","mode":"stop","state":true}]',
  1, NOW(), NOW()
);

INSERT INTO `acl` (
  `id`, `uid`, `name`, `des`, `default_action`, `data`, `enable`, `create_at`, `update_at`
) VALUES (
  10001, 1, 'cc-e2e-acl-default-deny',
  'allow single IP, default deny',
  'deny',
  '{"rules":[{"conditions":[{"item":"ip","operator":"eq","value":"10.0.0.60"}],"action":"allow"}],"default_deny_status":403}',
  1, NOW(), NOW()
),
(
  10002, 1, 'cc-e2e-acl-redirect',
  'redirect deny on /blocked',
  'allow',
  '{"rules":[{"conditions":[{"item":"uri","operator":"eq","value":"/blocked"}],"action":"deny","deny_status":302,"redirect_url":"http://origin:8080/redirected"}]}',
  1, NOW(), NOW()
);

INSERT INTO `site` (
  `id`, `uid`, `node_group_id`, `domain`, `backend`, `backend_protocol`,
  `cc_default_rule`, `state`, `enable`, `settings`, `create_at`, `update_at`
) VALUES (
  1, 1, 1,
  '["cc-test.local"]',
  '["origin:8080"]',
  'http',
  10001,
  'running',
  1,
  '{"access":{"acl":0},"security":{"custom_rules":[{"action":"allow","on":true,"matchers":[{"key":"uri","operator":"eq","value":"/api"}]},{"action":"block","on":false,"matchers":[{"key":"uri","operator":"eq","value":"/disabled-rule"}]}]}}',
  NOW(), NOW()
),
(
  2, 1, 1,
  '["shield-test.local"]',
  '["origin:8080"]',
  'http',
  10002,
  'running',
  1,
  '{"access":{"acl":0},"security":{"default_rule":10002,"custom_rules":[{"on":true,"breakMatch":true,"action":"5s","actionParams":{"seconds":10,"requests":0,"urlRequests":0,"blockOnFail":true},"matchers":[],"remark":"always-on 5s shield"}]}}',
  NOW(), NOW()
);
