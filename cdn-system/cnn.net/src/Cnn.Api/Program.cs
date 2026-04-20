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
using Cnn.Api;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddSqlSugar(builder.Configuration);
builder.Services.AddSignalR();
builder.Services.AddControllers();
builder.Services.AddRazorPages();
builder.Services.AddRazorComponents()
    .AddInteractiveServerComponents();
builder.Services.AddHttpContextAccessor();
builder.Services.AddMudServices();
builder.Services.AddApplicationServices();

var app = builder.Build();

using (var scope = app.Services.CreateScope())
{
    var db = scope.ServiceProvider.GetRequiredService<ISqlSugarClient>();
    RuntimeSchema.Ensure(db, app.Logger);
}

if (!app.Environment.IsDevelopment())
{
    app.UseExceptionHandler("/Error");
    app.UseHsts();
}

app.UseHttpsRedirection();
app.UseStaticFiles();
app.UseRouting();
app.UseWebSockets();
app.UseCors("AllowAll");
app.UseMiddleware<ApiAuthMiddleware>();
app.Use((context, next) =>
{
    if (string.IsNullOrWhiteSpace(context.TraceIdentifier))
    {
        context.TraceIdentifier = Guid.NewGuid().ToString("N");
    }
    return next();
});
app.UseMiddleware<MaintenanceMiddleware>();
app.UseAntiforgery();

app.MapApplicationEndpoints();

app.MapControllers();
app.MapHub<TaskHub>("/ws/tasks");
app.MapHub<AdminHub>("/ws/admin");
app.MapRazorPages();
app.MapRazorComponents<App>()
    .AddInteractiveServerRenderMode();

app.Run();
