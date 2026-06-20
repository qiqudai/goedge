using Cnn.Common.Contracts;

namespace Cnn.Api.Services.Common;

public interface IMessageService
{
    Task<ServiceResult<MessageListResult>> ListAdminAsync(MessageListQuery query, string language, CancellationToken cancellationToken);
    Task<ServiceResult<MessageListResult>> ListUserAsync(MessageListQuery query, long? userId, string language, CancellationToken cancellationToken);
    Task<ServiceResult<MessageUnreadResult>> GetUnreadAsync(long? userId, string language, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> MarkReadAsync(long? userId, long messageId, CancellationToken cancellationToken);
    Task<ServiceResult<MessageSubListResult>> ListSubscriptionsAsync(long? userId, string language, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateSubscriptionsAsync(long? userId, MessageSubUpdateRequest request, CancellationToken cancellationToken);
}
