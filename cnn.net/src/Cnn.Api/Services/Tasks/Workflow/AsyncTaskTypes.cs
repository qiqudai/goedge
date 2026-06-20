namespace Cnn.Api.Services.Tasks.Workflow;

public static class AsyncTaskTypes
{
    public const string SiteCreate = "SITE_CREATE";
    public const string SiteUpdate = "SITE_UPDATE";
    public const string SiteEnable = "SITE_ENABLE";
    public const string SiteDisable = "SITE_DISABLE";
    public const string SiteDelete = "SITE_DELETE";
    public const string SiteBatchDelete = "SITE_BATCH_DELETE";
    public const string SiteGroupDelete = "SITE_GROUP_DELETE";

    public const string NodeCreate = "NODE_CREATE";
    public const string NodeUpdate = "NODE_UPDATE";
    public const string NodeEnable = "NODE_ENABLE";
    public const string NodeDisable = "NODE_DISABLE";
    public const string NodeDelete = "NODE_DELETE";

    public const string LineGroupCreate = "LINE_GROUP_CREATE";
    public const string LineGroupUpdate = "LINE_GROUP_UPDATE";
    public const string LineGroupDelete = "LINE_GROUP_DELETE";

    public const string CertificateCreate = "CERT_CREATE";
    public const string CertificateUpdate = "CERT_UPDATE";
    public const string CertificateEnable = "CERT_ENABLE";
    public const string CertificateDisable = "CERT_DISABLE";
    public const string CertificateDelete = "CERT_DELETE";

    public const string SecurityRuleCreate = "SECURITY_RULE_CREATE";
    public const string SecurityRuleUpdate = "SECURITY_RULE_UPDATE";
    public const string SecurityRuleDisable = "SECURITY_RULE_DISABLE";
    public const string SecurityRuleDelete = "SECURITY_RULE_DELETE";
    public const string AclRuleDelete = "ACL_RULE_DELETE";

    public const string ProductPlanUpdate = "PLAN_UPDATE";
    public const string ProductPlanDelete = "PLAN_DELETE";

    public const string SubscriptionDelete = "SUBSCRIPTION_DELETE";

    public const string StreamCreate = "STREAM_CREATE";
    public const string StreamUpdate = "STREAM_UPDATE";
    public const string StreamEnable = "STREAM_ENABLE";
    public const string StreamDisable = "STREAM_DISABLE";
    public const string StreamDelete = "STREAM_DELETE";
    public const string StreamBatchDelete = "STREAM_BATCH_DELETE";
    public const string StreamGroupDelete = "STREAM_GROUP_DELETE";

    public const string CachePurge = "CACHE_PURGE";
    public const string CachePreheat = "CACHE_PREHEAT";
    public const string ConfigSync = "CONFIG_SYNC";

    public const string UserPurge = "USER_PURGE";
}
