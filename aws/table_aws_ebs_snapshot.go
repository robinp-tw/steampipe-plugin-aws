package aws

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/plugin"
	"github.com/turbot/steampipe-plugin-sdk/plugin/transform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func tableAwsEBSSnapshot(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "aws_ebs_snapshot",
		Description: "AWS EBS Snapshot",
		Get: &plugin.GetConfig{
			KeyColumns:        plugin.SingleColumn("snapshot_id"),
			ShouldIgnoreError: isNotFoundError([]string{"InvalidSnapshot.NotFound", "InvalidSnapshotID.Malformed"}),
			Hydrate:           getAwsEBSSnapshot,
		},
		List: &plugin.ListConfig{
			Hydrate: listAwsEBSSnapshots,
		},
		GetMatrixItem: BuildRegionList,
		Columns: awsRegionalColumns([]*plugin.Column{
			{
				Name:        "snapshot_id",
				Description: "The ID of the snapshot. Each snapshot receives a unique identifier when it is created.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "arn",
				Description: "The Amazon Resource Name (ARN) specifying the snapshot.",
				Type:        proto.ColumnType_STRING,
				Hydrate:     getEBSSnapshotARN,
				Transform:   transform.FromValue(),
			},
			{
				Name:        "state",
				Description: "The snapshot state.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "volume_size",
				Description: "The size of the volume, in GiB.",
				Type:        proto.ColumnType_INT,
			},
			{
				Name:        "volume_id",
				Description: "The ID of the volume that was used to create the snapshot. Snapshots created by the CopySnapshot action have an arbitrary volume ID that should not be used for any purpose.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "encrypted",
				Description: "Indicates whether the snapshot is encrypted.",
				Type:        proto.ColumnType_BOOL,
			},
			{
				Name:        "start_time",
				Description: "The time stamp when the snapshot was initiated.",
				Type:        proto.ColumnType_TIMESTAMP,
			},
			{
				Name:        "description",
				Description: "The description for the snapshot.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "kms_key_id",
				Description: "The Amazon Resource Name (ARN) of the AWS Key Management Service (AWS KMS) customer master key (CMK) that was used to protect the volume encryption key for the parent volume.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "data_encryption_key_id",
				Description: "The data encryption key identifier for the snapshot. This value is a unique identifier that corresponds to the data encryption key that was used to encrypt the original volume or snapshot copy. Because data encryption keys are inherited by volumes created from snapshots, and vice versa, if snapshots share the same data encryption key identifier, then they belong to the same volume/snapshot lineage.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "progress",
				Description: "The progress of the snapshot, as a percentage.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "state_message",
				Description: "Encrypted Amazon EBS snapshots are copied asynchronously. If a snapshot copy operation fails this field displays error state details to help you diagnose why the error occurred.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "owner_alias",
				Description: "The AWS owner alias, from an Amazon-maintained list (amazon). This is not the user-configured AWS account alias set using the IAM console.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "owner_id",
				Description: "The AWS account ID of the EBS snapshot owner.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "create_volume_permissions",
				Description: "The users and groups that have the permissions for creating volumes from the snapshot.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getAwsEBSSnapshotCreateVolumePermissions,
			},
			{
				Name:        "tags_src",
				Description: "A list of tags assigned to the snapshot.",
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("Tags"),
			},

			/// Standard columns for all tables
			{
				Name:        "title",
				Description: resourceInterfaceDescription("title"),
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("SnapshotId"),
			},
			{
				Name:        "tags",
				Description: resourceInterfaceDescription("tags"),
				Type:        proto.ColumnType_JSON,
				Transform:   transform.From(ec2SnapshotTurbotTags),
			},
			{
				Name:        "akas",
				Description: resourceInterfaceDescription("akas"),
				Type:        proto.ColumnType_JSON,
				Hydrate:     getEBSSnapshotARN,
				Transform:   transform.FromValue().Transform(transform.EnsureStringArray),
			},
		}),
	}
}

//// LIST FUNCTION

func listAwsEBSSnapshots(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := d.KeyColumnQualString(matrixKeyRegion)
	plugin.Logger(ctx).Trace("listAwsEBSSnapshots", "AWS_REGION", region)

	// Create session
	svc, err := Ec2Service(ctx, d, region)
	if err != nil {
		return nil, err
	}

	// List call
	err = svc.DescribeSnapshotsPages(
		&ec2.DescribeSnapshotsInput{
			OwnerIds: []*string{aws.String("self")},
		},
		func(page *ec2.DescribeSnapshotsOutput, isLast bool) bool {
			for _, snapshot := range page.Snapshots {
				d.StreamListItem(ctx, snapshot)

			}
			return !isLast
		},
	)

	return nil, err
}

//// HYDRATE FUNCTIONS

func getAwsEBSSnapshot(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getAwsEBSSnapshot")

	region := d.KeyColumnQualString(matrixKeyRegion)
	snapshotID := d.KeyColumnQuals["snapshot_id"].GetStringValue()

	// get service
	svc, err := Ec2Service(ctx, d, region)
	if err != nil {
		return nil, err
	}

	// Build the params
	params := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String(snapshotID)},
	}

	// Get call
	data, err := svc.DescribeSnapshots(params)
	if err != nil {
		plugin.Logger(ctx).Debug("getAwsEBSSnapshot__", "ERROR", err)
		return nil, err
	}

	if data.Snapshots != nil {
		return data.Snapshots[0], nil
	}
	return nil, nil
}

// getAwsEBSSnapshotCreateVolumePermissions :: Describes the users and groups that have the permissions for creating volumes from the snapshot
func getAwsEBSSnapshotCreateVolumePermissions(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getAwsEBSSnapshotCreateVolumePermissions")
	snapshotData := h.Item.(*ec2.Snapshot)
	region := d.KeyColumnQualString(matrixKeyRegion)

	// Create session
	svc, err := Ec2Service(ctx, d, region)
	if err != nil {
		return nil, err
	}

	// Build params
	params := &ec2.DescribeSnapshotAttributeInput{
		SnapshotId: snapshotData.SnapshotId,
		Attribute:  aws.String(ec2.SnapshotAttributeNameCreateVolumePermission),
	}

	// Describe create volume permission
	resp, err := svc.DescribeSnapshotAttribute(params)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func getEBSSnapshotARN(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getEBSSnapshotARN")
	region := d.KeyColumnQualString(matrixKeyRegion)
	snapshotData := h.Item.(*ec2.Snapshot)
	getCommonColumnsCached := plugin.HydrateFunc(getCommonColumns).WithCache()
	c, err := getCommonColumnsCached(ctx, d, h)
	if err != nil {
		return nil, err
	}
	commonColumnData := c.(*awsCommonColumnData)

	// Get the resource arn
	arn := "arn:" + commonColumnData.Partition + ":ec2:" + region + ":" + commonColumnData.AccountId + ":snapshot/" + *snapshotData.SnapshotId

	return arn, nil
}

//// TRANSFORM FUNCTIONS

func ec2SnapshotTurbotTags(_ context.Context, d *transform.TransformData) (interface{}, error) {
	snapshot := d.HydrateItem.(*ec2.Snapshot)
	return ec2TagsToMap(snapshot.Tags)
}
