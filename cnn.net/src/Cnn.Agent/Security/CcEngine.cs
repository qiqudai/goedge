using System.Net;
using System.Text.RegularExpressions;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Security;

public sealed class CcEngine
{
    private readonly CcCounterStore _counterStore = new(32);

    public bool TryEvaluate(HttpContext context, EdgeDomainDto domain, SecurityConfigSnapshot snapshot, out SecurityDecision decision)
    {
        decision = SecurityDecision.Allow();

        var groupId = domain.CcRuleId.GetValueOrDefault();
        if (groupId <= 0)
        {
            return false;
        }

        if (!snapshot.CcRulesByGroupId.TryGetValue(groupId, out var rules) || rules.Count == 0)
        {
            return false;
        }

        var ip = NormalizeIp(context.Connection.RemoteIpAddress);
        var host = context.Request.Host.Host?.Trim().ToLowerInvariant() ?? string.Empty;
        var path = context.Request.Path.ToString();
        var nowSec = DateTimeOffset.UtcNow.ToUnixTimeSeconds();

        foreach (var rule in rules)
        {
            if (!rule.Enabled)
            {
                continue;
            }

            if (!IsMatcherHit(context, rule.Matcher))
            {
                continue;
            }

            var action = NormalizeAction(rule.Action, rule.Filter?.Type);
            if (action == "allow")
            {
                continue;
            }

            if (rule.Filter == null)
            {
                decision = BuildActionDecision(action, rule.RuleId, "cc_match");
                return true;
            }

            var overLimit = false;
            var window = Math.Max(1, rule.Filter.WithinSecond);

            if (rule.Filter.MaxReq > 0)
            {
                var key = $"t|{window}|{ip}|{host}";
                var count = _counterStore.Increment(key, window, nowSec);
                if (count > rule.Filter.MaxReq)
                {
                    overLimit = true;
                }
            }

            if (!overLimit && rule.Filter.MaxReqPerUri > 0)
            {
                var uriKey = $"u|{window}|{ip}|{host}|{path}";
                var uriCount = _counterStore.Increment(uriKey, window, nowSec);
                if (uriCount > rule.Filter.MaxReqPerUri)
                {
                    overLimit = true;
                }
            }

            if (!overLimit)
            {
                continue;
            }

            decision = BuildActionDecision(action, rule.RuleId, "cc_limit");
            return true;
        }

        return false;
    }

    private static SecurityDecision BuildActionDecision(string action, string? ruleId, string reason)
    {
        return action switch
        {
            "limit_rate" => SecurityDecision.Block(StatusCodes.Status429TooManyRequests, "cc", ruleId, reason, "conn_limit"),
            "block" => SecurityDecision.Block(StatusCodes.Status403Forbidden, "cc", ruleId, reason, "403"),
            "deny" => SecurityDecision.Block(StatusCodes.Status403Forbidden, "cc", ruleId, reason, "403"),
            "captcha" => SecurityDecision.Block(StatusCodes.Status429TooManyRequests, "cc", ruleId, reason, "conn_limit"),
            "slide" => SecurityDecision.Block(StatusCodes.Status429TooManyRequests, "cc", ruleId, reason, "conn_limit"),
            "click" => SecurityDecision.Block(StatusCodes.Status429TooManyRequests, "cc", ruleId, reason, "conn_limit"),
            _ => SecurityDecision.Block(StatusCodes.Status403Forbidden, "cc", ruleId, reason, "403")
        };
    }

    private static bool IsMatcherHit(HttpContext context, CcMatcherCompiled matcher)
    {
        var clauses = matcher.Clauses;
        if (clauses == null || clauses.Count == 0)
        {
            return true;
        }

        foreach (var clause in clauses)
        {
            var actual = ResolveActualValue(context, clause);
            if (!Compare(actual, clause))
            {
                return false;
            }
        }

        return true;
    }

    private static string ResolveActualValue(HttpContext context, CcMatcherClause clause)
    {
        var item = clause.Item.Trim().ToLowerInvariant();
        return item switch
        {
            "ip" or "client_ip" or "remote_ip" => NormalizeIp(context.Connection.RemoteIpAddress),
            "host" or "domain" => context.Request.Host.Host?.Trim().ToLowerInvariant() ?? string.Empty,
            "uri" or "path" or "url" => context.Request.Path.ToString(),
            "query" => context.Request.QueryString.ToString(),
            "method" => context.Request.Method,
            "ua" or "user_agent" => context.Request.Headers.UserAgent.ToString(),
            "referer" or "referrer" => context.Request.Headers.Referer.ToString(),
            "header" => ResolveHeader(context, clause.HeaderName),
            _ => ResolveExtendedItem(context, item)
        };
    }

    private static string ResolveExtendedItem(HttpContext context, string item)
    {
        if (item.StartsWith("header:", StringComparison.OrdinalIgnoreCase))
        {
            var name = item.Substring("header:".Length).Trim();
            return ResolveHeader(context, name);
        }

        if (context.Request.Headers.TryGetValue(item, out var values))
        {
            return values.ToString();
        }

        return string.Empty;
    }

    private static string ResolveHeader(HttpContext context, string? headerName)
    {
        if (string.IsNullOrWhiteSpace(headerName))
        {
            return string.Empty;
        }

        if (!context.Request.Headers.TryGetValue(headerName.Trim(), out var values))
        {
            return string.Empty;
        }

        return values.ToString();
    }

    private static bool Compare(string actualRaw, CcMatcherClause clause)
    {
        var actual = actualRaw ?? string.Empty;
        var value = clause.Value ?? string.Empty;
        var op = (clause.Operator ?? "contains").Trim().ToLowerInvariant();

        if (op is "exists" or "present")
        {
            return !string.IsNullOrWhiteSpace(actual);
        }

        if (op is "not_exists" or "absent")
        {
            return string.IsNullOrWhiteSpace(actual);
        }

        if (string.Equals(clause.Item, "ip", StringComparison.OrdinalIgnoreCase) ||
            string.Equals(clause.Item, "client_ip", StringComparison.OrdinalIgnoreCase) ||
            string.Equals(clause.Item, "remote_ip", StringComparison.OrdinalIgnoreCase))
        {
            if (IPAddress.TryParse(actual, out var ip))
            {
                if (op is "eq" or "=" or "in")
                {
                    return IpRuleMatcher.IsMatch(ip, value);
                }

                if (op is "neq" or "!=")
                {
                    return !IpRuleMatcher.IsMatch(ip, value);
                }
            }
        }

        return op switch
        {
            "eq" or "=" or "equal" => string.Equals(actual, value, StringComparison.OrdinalIgnoreCase),
            "neq" or "!=" => !string.Equals(actual, value, StringComparison.OrdinalIgnoreCase),
            "contains" or "in" => actual.Contains(value, StringComparison.OrdinalIgnoreCase),
            "not_contains" => !actual.Contains(value, StringComparison.OrdinalIgnoreCase),
            "prefix" or "starts_with" or "^=" => actual.StartsWith(value, StringComparison.OrdinalIgnoreCase),
            "suffix" or "ends_with" or "$=" => actual.EndsWith(value, StringComparison.OrdinalIgnoreCase),
            "regex" or "re" or "~" => clause.Regex != null && SafeRegexIsMatch(clause.Regex, actual),
            "gt" => TryCompareNumber(actual, value, static (a, b) => a > b),
            "gte" or "ge" => TryCompareNumber(actual, value, static (a, b) => a >= b),
            "lt" => TryCompareNumber(actual, value, static (a, b) => a < b),
            "lte" or "le" => TryCompareNumber(actual, value, static (a, b) => a <= b),
            _ => actual.Contains(value, StringComparison.OrdinalIgnoreCase)
        };
    }

    private static bool SafeRegexIsMatch(Regex regex, string input)
    {
        try
        {
            return regex.IsMatch(input);
        }
        catch
        {
            return false;
        }
    }

    private static bool TryCompareNumber(string leftRaw, string rightRaw, Func<double, double, bool> comparer)
    {
        if (!double.TryParse(leftRaw, out var left))
        {
            return false;
        }

        if (!double.TryParse(rightRaw, out var right))
        {
            return false;
        }

        return comparer(left, right);
    }

    private static string NormalizeAction(string? action, string? fallback)
    {
        var value = string.IsNullOrWhiteSpace(action) ? fallback : action;
        if (string.IsNullOrWhiteSpace(value))
        {
            return "block";
        }

        return value.Trim().ToLowerInvariant();
    }

    private static string NormalizeIp(IPAddress? ip)
    {
        if (ip == null)
        {
            return string.Empty;
        }

        if (ip.IsIPv4MappedToIPv6)
        {
            return ip.MapToIPv4().ToString();
        }

        return ip.ToString();
    }

    private sealed class CcCounterStore
    {
        private readonly CounterShard[] _shards;

        public CcCounterStore(int shardCount)
        {
            if (shardCount < 16)
            {
                shardCount = 16;
            }

            _shards = Enumerable.Range(0, shardCount).Select(_ => new CounterShard()).ToArray();
        }

        public int Increment(string key, int windowSeconds, long nowUnixSeconds)
        {
            if (string.IsNullOrEmpty(key))
            {
                return 0;
            }

            var index = (int)((uint)StringComparer.Ordinal.GetHashCode(key) % (uint)_shards.Length);
            return _shards[index].Increment(key, windowSeconds, nowUnixSeconds);
        }

        private sealed class CounterShard
        {
            private readonly object _lock = new();
            private readonly Dictionary<string, CounterEntry> _entries = new(StringComparer.Ordinal);

            public int Increment(string key, int windowSeconds, long nowUnixSeconds)
            {
                var windowStart = nowUnixSeconds / Math.Max(1, windowSeconds);
                lock (_lock)
                {
                    if (!_entries.TryGetValue(key, out var entry))
                    {
                        entry = new CounterEntry
                        {
                            WindowStart = windowStart,
                            Count = 0,
                            LastSeen = nowUnixSeconds
                        };
                        _entries[key] = entry;
                    }
                    else if (entry.WindowStart != windowStart)
                    {
                        entry.WindowStart = windowStart;
                        entry.Count = 0;
                    }

                    entry.LastSeen = nowUnixSeconds;
                    entry.Count++;

                    if (_entries.Count > 200_000)
                    {
                        Cleanup(nowUnixSeconds);
                    }

                    return entry.Count;
                }
            }

            private void Cleanup(long nowUnixSeconds)
            {
                if (_entries.Count == 0)
                {
                    return;
                }

                var toRemove = new List<string>();
                foreach (var (key, entry) in _entries)
                {
                    if (nowUnixSeconds - entry.LastSeen > 600)
                    {
                        toRemove.Add(key);
                    }

                    if (toRemove.Count >= 4096)
                    {
                        break;
                    }
                }

                foreach (var key in toRemove)
                {
                    _entries.Remove(key);
                }
            }

            private sealed class CounterEntry
            {
                public long WindowStart { get; set; }
                public int Count { get; set; }
                public long LastSeen { get; set; }
            }
        }
    }
}
