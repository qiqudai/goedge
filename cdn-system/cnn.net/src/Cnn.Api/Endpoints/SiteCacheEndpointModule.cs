using Cnn.Api.Cache;
using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
namespace Cnn.Api.Endpoints;

public static class SiteCacheEndpointModule
{
    public static void Map(WebApplication app)
    {
        app.MapGet("/api/sites/{id:int}/cache", async (
            HttpContext context,
            IMessageLocalizer localizer,
            int id,
            ISiteCacheApplicationService cacheService
        ) =>
        {
            var result = await cacheService.GetAsync(id, context.RequestAborted);
            if (!result.Success)
            {
                return Results.NotFound(ApiResponseFactory.Fail<object>(context, localizer, result.ErrorCode == 0 ? ErrorCodes.NotFound : result.ErrorCode));
            }
            return Results.Ok(ApiResponseFactory.Ok(context, localizer, result.Data));
        });

        app.MapPost("/api/sites/{id:int}/cache", async (
            HttpContext context,
            IMessageLocalizer localizer,
            int id,
            CacheConfigDto input,
            bool compile,
            ISiteCacheApplicationService cacheService
        ) =>
        {
            var result = await cacheService.SaveAsync(id, input, compile, context.RequestAborted);
            if (!result.Success)
            {
                return Results.NotFound(ApiResponseFactory.Fail<object>(context, localizer, result.ErrorCode == 0 ? ErrorCodes.NotFound : result.ErrorCode));
            }
            return Results.Ok(ApiResponseFactory.Ok(context, localizer, result.Data));
        });

        app.MapPost("/api/sites/{id:int}/cache/compile", async (
            HttpContext context,
            IMessageLocalizer localizer,
            int id,
            ISiteCacheApplicationService cacheService
        ) =>
        {
            var result = await cacheService.CompileAsync(id, context.RequestAborted);
            if (!result.Success)
            {
                return Results.NotFound(ApiResponseFactory.Fail<object>(context, localizer, result.ErrorCode == 0 ? ErrorCodes.NotFound : result.ErrorCode));
            }
            return Results.Ok(ApiResponseFactory.Ok(context, localizer, result.Data));
        });
    }
}
