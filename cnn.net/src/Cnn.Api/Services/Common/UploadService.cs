using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Microsoft.AspNetCore.Hosting;

namespace Cnn.Api.Services.Common;

public interface IUploadService
{
    Task<ServiceResult<UploadImageResult>> SaveImageAsync(IFormFile? file, CancellationToken cancellationToken);
}

public sealed class UploadService : IUploadService
{
    private static readonly HashSet<string> AllowedExt = new(StringComparer.OrdinalIgnoreCase)
    {
        ".jpg", ".jpeg", ".png", ".ico", ".gif", ".webp"
    };

    private readonly IWebHostEnvironment _environment;

    public UploadService(IWebHostEnvironment environment)
    {
        _environment = environment;
    }

    public async Task<ServiceResult<UploadImageResult>> SaveImageAsync(IFormFile? file, CancellationToken cancellationToken)
    {
        if (file == null || file.Length == 0)
        {
            return ServiceResult<UploadImageResult>.Fail(ErrorCodes.MissingParam, "upload_file_required");
        }

        var ext = Path.GetExtension(file.FileName ?? string.Empty);
        if (string.IsNullOrWhiteSpace(ext) || !AllowedExt.Contains(ext))
        {
            return ServiceResult<UploadImageResult>.Fail(ErrorCodes.InvalidParam, "upload_image_only");
        }

        var root = _environment.WebRootPath;
        if (string.IsNullOrWhiteSpace(root))
        {
            root = Path.Combine(AppContext.BaseDirectory, "wwwroot");
        }

        var uploadDir = Path.Combine(root, "uploads", "images");
        Directory.CreateDirectory(uploadDir);

        var filename = $"{DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()}{ext.ToLowerInvariant()}";
        var filePath = Path.Combine(uploadDir, filename);

        try
        {
            await using var stream = new FileStream(filePath, FileMode.CreateNew, FileAccess.Write, FileShare.None, 4096, true);
            await file.CopyToAsync(stream, cancellationToken);
        }
        catch
        {
            return ServiceResult<UploadImageResult>.Fail(ErrorCodes.InternalError, "upload_save_failed");
        }

        return ServiceResult<UploadImageResult>.Ok(new UploadImageResult
        {
            Url = $"/uploads/images/{filename}"
        });
    }
}
