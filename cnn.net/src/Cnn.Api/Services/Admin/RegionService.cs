using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class RegionService : IRegionService
{
    private readonly ISqlSugarClient _db;
    private readonly RegionMetaService _metaService;

    public RegionService(ISqlSugarClient db)
    {
        _db = db;
        _metaService = new RegionMetaService(db);
    }

    public async Task<ServiceResult<RegionListResult>> ListAsync(CancellationToken cancellationToken)
    {
        var regions = await _db.Queryable<Region>().OrderBy(r => r.Id, OrderByType.Asc).ToListAsync();
        var metaMap = await _metaService.LoadAsync();

        var list = regions.Select(region =>
        {
            var l2Port = RegionMetaService.ResolveL2CheckPort(metaMap, region.Id);
            var sortOrder = RegionMetaService.ResolveSortOrder(metaMap, region.Id);
            return new RegionListItem(
                region.Id,
                region.Name,
                region.Des,
                l2Port,
                sortOrder,
                region.CreateAt,
                region.UpdateAt
            );
        }).ToList();

        return ServiceResult<RegionListResult>.Ok(new RegionListResult(list, list.Count));
    }

    public async Task<ServiceResult<bool>> CreateAsync(RegionUpsertRequest request, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(request.Name))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var l2Port = request.L2CheckPort.GetValueOrDefault() == 0 ? 80 : request.L2CheckPort!.Value;
        var sortOrder = request.SortOrder.GetValueOrDefault() == 0 ? 100 : request.SortOrder!.Value;

        var now = DateTime.Now;
        var region = new Region
        {
            Name = request.Name.Trim(),
            Des = request.Remark?.Trim(),
            CreateAt = now,
            UpdateAt = now
        };

        var id = await _db.Insertable(region).ExecuteReturnIdentityAsync();
        region.Id = id;

        var metaMap = await _metaService.LoadAsync();
        metaMap[id.ToString()] = new RegionMeta { L2CheckPort = l2Port, SortOrder = sortOrder };
        var saved = await _metaService.SaveAsync(metaMap);
        if (!saved)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InternalError, "save_failed");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long regionId, RegionUpsertRequest request, CancellationToken cancellationToken)
    {
        if (regionId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        if (string.IsNullOrWhiteSpace(request.Name))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var l2Port = request.L2CheckPort.GetValueOrDefault() == 0 ? 80 : request.L2CheckPort!.Value;
        var sortOrder = request.SortOrder.GetValueOrDefault() == 0 ? 100 : request.SortOrder!.Value;

        var name = request.Name.Trim();
        var remark = request.Remark?.Trim();
        var now = DateTime.Now;

        await _db.Updateable<Region>()
            .SetColumns(r => new Region
            {
                Name = name,
                Des = remark,
                UpdateAt = now
            })
            .Where(r => r.Id == regionId)
            .ExecuteCommandAsync();

        var metaMap = await _metaService.LoadAsync();
        metaMap[regionId.ToString()] = new RegionMeta { L2CheckPort = l2Port, SortOrder = sortOrder };
        var saved = await _metaService.SaveAsync(metaMap);
        if (!saved)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InternalError, "save_failed");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long regionId, CancellationToken cancellationToken)
    {
        if (regionId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var nodeCount = await _db.Queryable<Node>().Where(n => n.RegionId == regionId).CountAsync();
        if (nodeCount > 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InUse, "region.has_nodes");
        }

        await _db.Deleteable<Region>().Where(r => r.Id == regionId).ExecuteCommandAsync();

        var metaMap = await _metaService.LoadAsync();
        metaMap.Remove(regionId.ToString());
        var saved = await _metaService.SaveAsync(metaMap);
        if (!saved)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InternalError, "save_failed");
        }

        return ServiceResult<bool>.Ok(true);
    }
}
