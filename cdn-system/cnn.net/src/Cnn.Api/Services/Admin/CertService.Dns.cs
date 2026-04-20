using System.Net;
using System.Net.NetworkInformation;
using System.Net.Sockets;
using System.Text;
using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class CertService
{
    public async Task<ServiceResult<DnsChallengeInfoDto?>> GetDnsChallengeAsync(
        long id,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<DnsChallengeInfoDto?>.Fail(ErrorCodes.InvalidParam, "cert_invalid_id");
        }

        var certQuery = _db.Queryable<Cert>().Where(c => c.Id == id);
        if (!isAdmin)
        {
            var uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<DnsChallengeInfoDto?>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            certQuery = certQuery.Where(c => c.Uid == (int)uid);
        }

        var cert = await certQuery.FirstAsync();
        if (cert == null)
        {
            return ServiceResult<DnsChallengeInfoDto?>.Fail(ErrorCodes.NotFound, "cert_not_found");
        }

        if (string.IsNullOrWhiteSpace(cert.Ret))
        {
            return ServiceResult<DnsChallengeInfoDto?>.Ok(null);
        }

        try
        {
            var info = JsonSerializer.Deserialize<DnsChallengeInfoDto>(cert.Ret, JsonOptions);
            return ServiceResult<DnsChallengeInfoDto?>.Ok(info);
        }
        catch
        {
            return ServiceResult<DnsChallengeInfoDto?>.Ok(null);
        }
    }

    public async Task<ServiceResult<bool>> VerifyDnsChallengeAsync(
        long id,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "cert_invalid_id");
        }

        var certQuery = _db.Queryable<Cert>().Where(c => c.Id == id);
        if (!isAdmin)
        {
            var uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            certQuery = certQuery.Where(c => c.Uid == (int)uid);
        }

        var cert = await certQuery.FirstAsync();
        if (cert == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "cert_not_found");
        }

        DnsChallengeInfoDto? info;
        try
        {
            info = JsonSerializer.Deserialize<DnsChallengeInfoDto>(cert.Ret ?? string.Empty, JsonOptions);
        }
        catch
        {
            info = null;
        }

        if (info == null || string.IsNullOrWhiteSpace(info.Fqdn) || string.IsNullOrWhiteSpace(info.RecordValue))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "dns_challenge_not_found");
        }

        var ok = await LookupTxtRecordAsync(info.Fqdn, info.RecordValue, cancellationToken);
        if (!ok)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "dns_txt_not_found");
        }

        return ServiceResult<bool>.Ok(true);
    }

    private static async Task<bool> LookupTxtRecordAsync(string fqdn, string expected, CancellationToken cancellationToken)
    {
        fqdn = fqdn.Trim().TrimEnd('.');
        expected = expected.Trim();
        if (string.IsNullOrWhiteSpace(fqdn) || string.IsNullOrWhiteSpace(expected))
        {
            return false;
        }

        try
        {
            var records = await QueryTxtAsync(fqdn, cancellationToken);
            return records.Any(record => string.Equals(record.Trim(), expected, StringComparison.Ordinal));
        }
        catch
        {
            return false;
        }
    }

    internal static async Task<IReadOnlyList<string>> QueryTxtAsync(string fqdn, CancellationToken cancellationToken)
    {
        var server = ResolveDnsServer();
        if (server == null)
        {
            return Array.Empty<string>();
        }

        var request = BuildDnsQuery(fqdn);
        using var udp = new UdpClient();
        udp.Client.ReceiveTimeout = 5000;
        udp.Client.SendTimeout = 5000;

        await udp.SendAsync(request, request.Length, new IPEndPoint(server, 53));
        var receiveTask = udp.ReceiveAsync();
        var completed = await Task.WhenAny(receiveTask, Task.Delay(TimeSpan.FromSeconds(5), cancellationToken));
        if (completed != receiveTask)
        {
            return Array.Empty<string>();
        }

        var response = receiveTask.Result.Buffer;
        if (response.Length < 12)
        {
            return Array.Empty<string>();
        }

        return ParseTxtResponse(response);
    }

    private static IPAddress? ResolveDnsServer()
    {
        foreach (var nic in NetworkInterface.GetAllNetworkInterfaces())
        {
            if (nic.OperationalStatus != OperationalStatus.Up)
            {
                continue;
            }

            var props = nic.GetIPProperties();
            foreach (var address in props.DnsAddresses)
            {
                if (address.AddressFamily == AddressFamily.InterNetwork)
                {
                    return address;
                }
            }
        }

        return IPAddress.Parse("8.8.8.8");
    }

    private static byte[] BuildDnsQuery(string fqdn)
    {
        var id = (ushort)Random.Shared.Next(0, ushort.MaxValue + 1);
        var buffer = new List<byte>();

        void WriteUShort(ushort value)
        {
            buffer.Add((byte)(value >> 8));
            buffer.Add((byte)(value & 0xff));
        }

        WriteUShort(id);
        WriteUShort(0x0100);
        WriteUShort(1);
        WriteUShort(0);
        WriteUShort(0);
        WriteUShort(0);

        foreach (var part in fqdn.Split('.', StringSplitOptions.RemoveEmptyEntries))
        {
            var bytes = Encoding.UTF8.GetBytes(part);
            buffer.Add((byte)bytes.Length);
            buffer.AddRange(bytes);
        }
        buffer.Add(0);

        WriteUShort(16);
        WriteUShort(1);

        return buffer.ToArray();
    }

    private static IReadOnlyList<string> ParseTxtResponse(byte[] response)
    {
        if (response.Length < 12)
        {
            return Array.Empty<string>();
        }

        var qdCount = ReadUShort(response, 4);
        var anCount = ReadUShort(response, 6);
        var offset = 12;

        for (var i = 0; i < qdCount; i++)
        {
            SkipName(response, ref offset);
            offset += 4;
            if (offset >= response.Length)
            {
                return Array.Empty<string>();
            }
        }

        var records = new List<string>();
        for (var i = 0; i < anCount; i++)
        {
            SkipName(response, ref offset);
            if (offset + 10 > response.Length)
            {
                return records;
            }

            var type = ReadUShort(response, offset);
            offset += 2;
            offset += 2;
            offset += 4;
            var rdLength = ReadUShort(response, offset);
            offset += 2;

            if (offset + rdLength > response.Length)
            {
                return records;
            }

            if (type == 16)
            {
                var end = offset + rdLength;
                while (offset < end)
                {
                    var len = response[offset++];
                    if (len == 0 || offset + len > end)
                    {
                        break;
                    }
                    var text = Encoding.UTF8.GetString(response, offset, len);
                    records.Add(text);
                    offset += len;
                }
            }
            else
            {
                offset += rdLength;
            }
        }

        return records;
    }

    private static void SkipName(byte[] message, ref int offset)
    {
        while (offset < message.Length)
        {
            var len = message[offset++];
            if (len == 0)
            {
                return;
            }
            if ((len & 0xC0) == 0xC0)
            {
                offset++;
                return;
            }
            offset += len;
        }
    }

    private static ushort ReadUShort(byte[] message, int offset)
    {
        if (offset + 1 >= message.Length)
        {
            return 0;
        }
        return (ushort)((message[offset] << 8) | message[offset + 1]);
    }
}


