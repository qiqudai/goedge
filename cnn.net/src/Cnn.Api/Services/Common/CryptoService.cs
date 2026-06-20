using System.Security.Cryptography;
using System.Text;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Common;

public interface ICryptoService
{
    string? Encrypt(string plain);
    string? Decrypt(string cipherText);
}

public sealed class CryptoService : ICryptoService
{
    private const string DefaultSecretKey = "0123456789abcdef0123456789abcdef";
    private const int AesGcmTagSize = 16;
    private readonly IConfiguration _configuration;

    public CryptoService(IConfiguration configuration)
    {
        _configuration = configuration;
    }

    public string? Encrypt(string plain)
    {
        if (string.IsNullOrEmpty(plain))
        {
            return string.Empty;
        }

        var key = ResolveKey();
        if (key == null)
        {
            return null;
        }

        try
        {
            using var aes = new AesGcm(key, AesGcmTagSize);
            var nonce = RandomNumberGenerator.GetBytes(AesGcm.NonceByteSizes.MaxSize);
            var plainBytes = Encoding.UTF8.GetBytes(plain);
            var cipherBytes = new byte[plainBytes.Length];
            var tag = new byte[AesGcmTagSize];

            aes.Encrypt(nonce, plainBytes, cipherBytes, tag);

            var output = new byte[nonce.Length + tag.Length + cipherBytes.Length];
            Buffer.BlockCopy(nonce, 0, output, 0, nonce.Length);
            Buffer.BlockCopy(tag, 0, output, nonce.Length, tag.Length);
            Buffer.BlockCopy(cipherBytes, 0, output, nonce.Length + tag.Length, cipherBytes.Length);
            return Convert.ToBase64String(output);
        }
        catch
        {
            return null;
        }
    }

    public string? Decrypt(string cipherText)
    {
        if (string.IsNullOrEmpty(cipherText))
        {
            return string.Empty;
        }

        var key = ResolveKey();
        if (key == null)
        {
            return null;
        }

        try
        {
            var input = Convert.FromBase64String(cipherText);
            var nonceSize = AesGcm.NonceByteSizes.MaxSize;
            var tagSize = AesGcmTagSize;
            if (input.Length <= nonceSize + tagSize)
            {
                return null;
            }

            var nonce = new byte[nonceSize];
            var tag = new byte[tagSize];
            var cipherBytes = new byte[input.Length - nonceSize - tagSize];

            Buffer.BlockCopy(input, 0, nonce, 0, nonceSize);
            Buffer.BlockCopy(input, nonceSize, tag, 0, tagSize);
            Buffer.BlockCopy(input, nonceSize + tagSize, cipherBytes, 0, cipherBytes.Length);

            using var aes = new AesGcm(key, tagSize);
            var plainBytes = new byte[cipherBytes.Length];
            aes.Decrypt(nonce, cipherBytes, tag, plainBytes);
            return Encoding.UTF8.GetString(plainBytes);
        }
        catch
        {
            return null;
        }
    }

    private byte[]? ResolveKey()
    {
        var raw = _configuration["App:SecretKey"];
        if (string.IsNullOrWhiteSpace(raw))
        {
            raw = DefaultSecretKey;
        }

        var bytes = Encoding.UTF8.GetBytes(raw);
        if (bytes.Length == 0)
        {
            return null;
        }

        if (bytes.Length == 32)
        {
            return bytes;
        }

        var padded = new byte[32];
        var length = Math.Min(bytes.Length, padded.Length);
        Buffer.BlockCopy(bytes, 0, padded, 0, length);
        return padded;
    }
}
