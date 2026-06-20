using System.Security.Cryptography;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Auth;

public interface ICaptchaService
{
    string GenerateCode(int length);
    Task<bool> StoreAsync(string? email, string? phone, string? ip, string code, CancellationToken cancellationToken);
    Task<bool> VerifyAsync(string? email, string? phone, string? code, CancellationToken cancellationToken);
}

public sealed class CaptchaService : ICaptchaService
{
    private static readonly TimeSpan CaptchaTtl = TimeSpan.FromMinutes(5);
    private readonly ISqlSugarClient _db;

    public CaptchaService(ISqlSugarClient db)
    {
        _db = db;
    }

    public string GenerateCode(int length)
    {
        if (length <= 0)
        {
            length = 6;
        }

        var bytes = new byte[length];
        try
        {
            RandomNumberGenerator.Fill(bytes);
        }
        catch
        {
            var seed = DateTime.UtcNow.Ticks;
            for (var i = 0; i < length; i++)
            {
                bytes[i] = (byte)('0' + seed % 10);
                seed /= 10;
                if (seed == 0)
                {
                    seed = DateTime.UtcNow.Ticks;
                }
            }
        }

        var chars = new char[length];
        for (var i = 0; i < length; i++)
        {
            chars[i] = (char)('0' + (bytes[i] % 10));
        }

        return new string(chars);
    }

    public async Task<bool> StoreAsync(string? email, string? phone, string? ip, string code, CancellationToken cancellationToken)
    {
        var record = new Captcha
        {
            Email = email?.Trim(),
            Phone = phone?.Trim(),
            CaptchaCode = code.Trim(),
            Ip = ip?.Trim(),
            CreateAt = DateTime.Now
        };

        var rows = await _db.Insertable(record).ExecuteCommandAsync();
        return rows > 0;
    }

    public async Task<bool> VerifyAsync(string? email, string? phone, string? code, CancellationToken cancellationToken)
    {
        code = code?.Trim();
        if (string.IsNullOrWhiteSpace(code))
        {
            return false;
        }

        var deadline = DateTime.Now.Subtract(CaptchaTtl);
        var query = _db.Queryable<Captcha>()
            .Where(c => c.CaptchaCode == code && c.CreateAt >= deadline);

        email = email?.Trim();
        phone = phone?.Trim();
        if (!string.IsNullOrWhiteSpace(email))
        {
            query = query.Where(c => c.Email == email);
        }
        else if (!string.IsNullOrWhiteSpace(phone))
        {
            query = query.Where(c => c.Phone == phone);
        }
        else
        {
            return false;
        }

        var count = await query.CountAsync();
        if (count == 0)
        {
            return false;
        }

        var deleteQuery = _db.Deleteable<Captcha>()
            .Where(c => c.CaptchaCode == code && c.CreateAt >= deadline);
        if (!string.IsNullOrWhiteSpace(email))
        {
            deleteQuery = deleteQuery.Where(c => c.Email == email);
        }
        else if (!string.IsNullOrWhiteSpace(phone))
        {
            deleteQuery = deleteQuery.Where(c => c.Phone == phone);
        }
        await deleteQuery.ExecuteCommandAsync();
        return true;
    }
}
