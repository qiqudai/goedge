using System.Text.Json;
using SqlSugar;

namespace Cnn.Api.Tests;

internal sealed class RealMySqlTestScope : IDisposable
{
    public RealMySqlTestScope()
    {
        var connectionString = ResolveConnectionString();
        Db = new SqlSugarClient(new ConnectionConfig
        {
            DbType = DbType.MySql,
            ConnectionString = connectionString,
            IsAutoCloseConnection = false,
            InitKeyType = InitKeyType.Attribute,
            ConfigureExternalServices = new ConfigureExternalServices
            {
                EntityService = (prop, column) =>
                {
                    if (column.IsIgnore)
                    {
                        return;
                    }

                    var desired = ToSnakeCase(prop.Name);
                    if (string.IsNullOrWhiteSpace(column.DbColumnName) ||
                        string.Equals(column.DbColumnName, prop.Name, StringComparison.OrdinalIgnoreCase))
                    {
                        column.DbColumnName = desired;
                    }
                }
            }
        });

        Db.Ado.Open();
        Db.Ado.BeginTran();
    }

    public SqlSugarClient Db { get; }

    public void Dispose()
    {
        try
        {
            Db.Ado.RollbackTran();
        }
        finally
        {
            Db.Dispose();
        }
    }

    private static string ResolveConnectionString()
    {
        var fromEnv = Environment.GetEnvironmentVariable("ConnectionStrings__Default");
        if (!string.IsNullOrWhiteSpace(fromEnv))
        {
            return fromEnv;
        }

        var appSettingsPath = "/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/appsettings.json";
        using var stream = File.OpenRead(appSettingsPath);
        using var document = JsonDocument.Parse(stream);
        if (document.RootElement.TryGetProperty("ConnectionStrings", out var section) &&
            section.TryGetProperty("Default", out var value) &&
            !string.IsNullOrWhiteSpace(value.GetString()))
        {
            return value.GetString()!;
        }

        throw new InvalidOperationException("MySQL connection string was not found.");
    }

    private static string ToSnakeCase(string name)
    {
        if (string.IsNullOrWhiteSpace(name))
        {
            return name;
        }

        var buffer = new System.Text.StringBuilder();
        for (var i = 0; i < name.Length; i++)
        {
            var c = name[i];
            if (char.IsUpper(c))
            {
                if (i > 0)
                {
                    buffer.Append('_');
                }
                buffer.Append(char.ToLowerInvariant(c));
            }
            else
            {
                buffer.Append(c);
            }
        }

        return buffer.ToString();
    }
}
