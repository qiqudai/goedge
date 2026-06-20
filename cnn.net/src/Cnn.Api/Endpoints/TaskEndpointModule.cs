using TaskEntity = Cnn.Domain.Entities.Task;
using SiteEntity = Cnn.Domain.Entities.Site;
using SiteConfCacheEntity = Cnn.Domain.Entities.SiteConfCache;
using ConfigEntity = Cnn.Domain.Entities.Config;
using Cnn.Api.Cache;
using Cnn.Common.Contracts;
using Cnn.Infrastructure.Db;
using Cnn.Api.Hubs;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Data;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Auth;
using Cnn.Api.Services.Authz;
using Cnn.Api.Services.Stats;
using Cnn.Api.Middleware;
using Microsoft.AspNetCore.SignalR;
using System.Net.WebSockets;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using SqlSugar;
using MudBlazor.Services;
using Cnn.Api.Extensions;

namespace Cnn.Api.Endpoints;

public static class TaskEndpointModule
{
    public static void Map(WebApplication app)
    {
        var jsonOptions = new JsonSerializerOptions(JsonSerializerDefaults.Web)
        {
            PropertyNameCaseInsensitive = true,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        };
        app.MapGet("/api/tasks", async (HttpContext context, IMessageLocalizer localizer, ISqlSugarClient db, int page = 1, int pageSize = 20) =>
        {
            if (page < 1) page = 1;
            if (pageSize < 1) pageSize = 20;
        
            var total = await db.Queryable<TaskEntity>().CountAsync();
            var list = await db.Queryable<TaskEntity>()
                .OrderBy(t => t.Id, OrderByType.Desc)
                .Skip((page - 1) * pageSize)
                .Take(pageSize)
                .Select(t => new TaskListItemDto
                {
                    Id = t.Id,
                    Pid = t.Pid,
                    Pry = t.Pry,
                    Name = t.Name,
                    Data = t.Data,
                    Type = t.Type,
                    Depend = t.Depend,
                    CreateAt = t.CreateAt,
                    StartAt = t.StartAt,
                    EndAt = t.EndAt,
                    State = t.State,
                    ErrTimes = t.ErrTimes,
                    RetryAt = t.RetryAt,
                    Ret = t.Ret,
                    TargetsJson = t.TargetsJson,
                    Progress = t.Progress
                })
                .ToListAsync();
        
            var payload = new { list, total, page };
            return Results.Ok(ApiResponseFactory.Ok(context, localizer, payload));
        });
        
        app.MapPost("/api/tasks/{id:long}/state", async (
            HttpContext context,
            IMessageLocalizer localizer,
            long id,
            TaskUpdateDto input,
            ISqlSugarClient db,
            IHubContext<TaskHub> hub,
            IAdminEventPublisher eventPublisher
        ) =>
        {
            var task = await db.Queryable<TaskEntity>().Where(t => t.Id == id).FirstAsync();
            if (task == null)
            {
                return Results.NotFound(ApiResponseFactory.Fail<object>(context, localizer, ErrorCodes.NotFound));
            }
        
            task.State = input.State ?? task.State;
            task.Progress = input.Progress ?? task.Progress;
            task.Ret = input.Ret ?? task.Ret;
            task.StartAt = input.StartAt ?? task.StartAt;
            task.EndAt = input.EndAt ?? task.EndAt;
        
            await db.Updateable(task).ExecuteCommandAsync();
        
            var update = new TaskUpdateDto(task.Id, task.State, task.Progress, task.Ret, task.StartAt, task.EndAt);
            await hub.Clients.All.SendAsync("task_update", update);
            await eventPublisher.PublishToAdminsAsync("task.state.changed", new
            {
                task_id = task.Id,
                type = task.Type,
                state = task.State,
                progress = task.Progress,
                ret = task.Ret,
                updated_at = DateTime.Now.ToString("yyyy-MM-dd HH:mm:ss")
            }, context.RequestAborted);
        
            return Results.Ok(ApiResponseFactory.Ok<object>(context, localizer, null));
        });
        
        
        

    }

}
