using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using SqlSugar;

namespace Cnn.Infrastructure.Db;

public static class SqlSugarSetup
{
    public static IServiceCollection AddSqlSugar(this IServiceCollection services, IConfiguration configuration)
    {
        var connectionString = configuration.GetConnectionString("Default") ?? string.Empty;
        var dbType = ResolveDbType(configuration, connectionString);
        var config = new ConnectionConfig
        {
            ConnectionString = connectionString,
            DbType = dbType,
            IsAutoCloseConnection = true,
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
        };

        var scope = new SqlSugarScope(config);
        services.AddSingleton<ISqlSugarClient>(scope);
        return services;
    }

    private static DbType ResolveDbType(IConfiguration configuration, string connectionString)
    {
        var provider = configuration["Database:Provider"]?.Trim();
        if (!string.IsNullOrWhiteSpace(provider))
        {
            if (provider.Equals("sqlite", StringComparison.OrdinalIgnoreCase))
            {
                return DbType.Sqlite;
            }
            if (provider.Equals("mysql", StringComparison.OrdinalIgnoreCase))
            {
                return DbType.MySql;
            }
        }

        var cs = connectionString?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(cs))
        {
            return DbType.MySql;
        }

        if (cs.StartsWith("Data Source=", StringComparison.OrdinalIgnoreCase) ||
            cs.StartsWith("DataSource=", StringComparison.OrdinalIgnoreCase) ||
            cs.StartsWith("Filename=", StringComparison.OrdinalIgnoreCase) ||
            cs.StartsWith("FileName=", StringComparison.OrdinalIgnoreCase))
        {
            return DbType.Sqlite;
        }

        return DbType.MySql;
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
