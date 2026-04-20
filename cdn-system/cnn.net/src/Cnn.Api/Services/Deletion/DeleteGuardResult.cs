namespace Cnn.Api.Services.Deletion;

public sealed class DeleteGuardResult
{
    public bool CanDelete { get; init; }
    public string? ErrorCode { get; init; }
    public string? Message { get; init; }
    public IReadOnlyList<DeleteReferenceItem> References { get; init; } = Array.Empty<DeleteReferenceItem>();

    public static DeleteGuardResult Allow()
    {
        return new DeleteGuardResult
        {
            CanDelete = true
        };
    }

    public static DeleteGuardResult Deny(
        string errorCode,
        string message,
        IReadOnlyList<DeleteReferenceItem>? references = null)
    {
        return new DeleteGuardResult
        {
            CanDelete = false,
            ErrorCode = errorCode,
            Message = message,
            References = references ?? Array.Empty<DeleteReferenceItem>()
        };
    }
}
