using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("node")]
public class Node
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int Pid { get; set; }

    public int? RegionId { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public string? Ip { get; set; }

    public string? Token { get; set; }

    public string? Host { get; set; }

    public int? Port { get; set; }

    public string? HttpProxy { get; set; }

    public bool? IsMgmt { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public bool? Enable { get; set; }

    public string? DisableBy { get; set; }

    public string? ConfigTask { get; set; }

    public bool? CheckOn { get; set; }

    public string? CheckProtocol { get; set; }

    public int? CheckTimeout { get; set; }

    public int? CheckPort { get; set; }

    public string? CheckHost { get; set; }

    public string? CheckPath { get; set; }

    public string? CheckNodeGroup { get; set; }

    public string? CheckAction { get; set; }

    public string? BwLimit { get; set; }

    public int? Level { get; set; }

    public int? Sort { get; set; }

    public string? CacheDir { get; set; }

    public int? MaxCacheSize { get; set; }

    public string? LogDir { get; set; }

    public string? SshHost { get; set; }

    public int? SshPort { get; set; }

    public string? SshUser { get; set; }

    public string? SshAuthType { get; set; }

    public string? SshPassword { get; set; }

    public string? SshKey { get; set; }

    public string? WorkDir { get; set; }

    public bool? AutoInstall { get; set; }

    public string? InstallStatus { get; set; }

    public string? InstallError { get; set; }

    public DateTime? InstallAt { get; set; }
}