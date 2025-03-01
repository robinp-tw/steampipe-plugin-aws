package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/plugin"
	"github.com/turbot/steampipe-plugin-sdk/plugin/transform"
)

//// TABLE DEFINITION
func tableAwsRdsInstanceMetricConnectionsHourly(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "aws_rds_db_instance_metric_connections_hourly",
		Description: "AWS RDS DB Instance Cloudwatch Metrics - DB Connections (Hourly)",
		List: &plugin.ListConfig{
			ParentHydrate: listRDSDBInstances,
			Hydrate:       listRdsInstanceMetricConnectionsHourly,
		},
		GetMatrixItem: BuildRegionList,
		Columns: awsRegionalColumns(cwMetricColumns(
			[]*plugin.Column{
				{
					Name:        "db_instance_identifier",
					Description: "The friendly name to identify the DB Instance.",
					Type:        proto.ColumnType_STRING,
					Transform:   transform.FromField("DimensionValue"),
				},
			})),
	}
}

func listRdsInstanceMetricConnectionsHourly(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	instance := h.Item.(*rds.DBInstance)
	return listCWMetricStatistics(ctx, d, "HOURLY", "AWS/RDS", "DatabaseConnections", "DBInstanceIdentifier", *instance.DBInstanceIdentifier)
}
