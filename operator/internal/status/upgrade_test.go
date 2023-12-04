package status

import (
	"context"
	"github.com/google/go-cmp/cmp"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestGenerateSchemaUpgrade(t *testing.T) {
	testNow := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	testEffectiveDate := lokiv1.StorageSchemaEffectiveDate("2023-12-06")
	testApplyTime := metav1.NewTime(testNow.Add(applySchemaDuration))

	tt := []struct {
		desc        string
		stack       *lokiv1.LokiStack
		now         time.Time
		wantUpgrade *lokiv1.ProposedSchemaUpdate
	}{
		{
			desc: "no upgrade - inuse current",
			stack: &lokiv1.LokiStack{Status: lokiv1.LokiStackStatus{
				Storage: lokiv1.LokiStackStorageStatus{
					Schemas: []lokiv1.ObjectStorageSchemaStatus{
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV13,
								EffectiveDate: "2023-01-01",
							},
							EndDate: "",
							Status:  lokiv1.SchemaStatusInUse,
						},
					},
				},
			}},
			now:         testNow,
			wantUpgrade: nil,
		},
		{
			desc: "no upgrade - future current",
			stack: &lokiv1.LokiStack{Status: lokiv1.LokiStackStatus{
				Storage: lokiv1.LokiStackStorageStatus{
					Schemas: []lokiv1.ObjectStorageSchemaStatus{
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV12,
								EffectiveDate: "2023-01-01",
							},
							EndDate: "2023-12-05",
							Status:  lokiv1.SchemaStatusInUse,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV13,
								EffectiveDate: "2023-12-06",
							},
							EndDate: "",
							Status:  lokiv1.SchemaStatusFuture,
						},
					},
				},
			}},
			now:         testNow,
			wantUpgrade: nil,
		},
		{
			desc: "remove obsolete",
			stack: &lokiv1.LokiStack{Status: lokiv1.LokiStackStatus{
				Storage: lokiv1.LokiStackStorageStatus{
					Schemas: []lokiv1.ObjectStorageSchemaStatus{
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV13,
								EffectiveDate: "2022-12-01",
							},
							EndDate: "2022-12-31",
							Status:  lokiv1.SchemaStatusObsolete,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV13,
								EffectiveDate: "2023-01-01",
							},
							EndDate: "",
							Status:  lokiv1.SchemaStatusInUse,
						},
					},
				},
			}},
			now: testNow,
			wantUpgrade: &lokiv1.ProposedSchemaUpdate{
				UpgradeTime: testApplyTime,
				Schemas: []lokiv1.ObjectStorageSchema{
					{
						Version:       upgradeSchemaVersion,
						EffectiveDate: "2023-01-01",
					},
				},
			},
		},
		{
			desc: "simple upgrade",
			stack: &lokiv1.LokiStack{Status: lokiv1.LokiStackStatus{
				Storage: lokiv1.LokiStackStorageStatus{
					Schemas: []lokiv1.ObjectStorageSchemaStatus{
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV12,
								EffectiveDate: "2023-01-01",
							},
							EndDate: "",
							Status:  lokiv1.SchemaStatusInUse,
						},
					},
				},
			}},
			now: testNow,
			wantUpgrade: &lokiv1.ProposedSchemaUpdate{
				UpgradeTime: testApplyTime,
				Schemas: []lokiv1.ObjectStorageSchema{
					{
						Version:       lokiv1.ObjectStorageSchemaV12,
						EffectiveDate: "2023-01-01",
					},
					{
						Version:       upgradeSchemaVersion,
						EffectiveDate: testEffectiveDate,
					},
				},
			},
		},
		{
			desc: "upgrade - replace single future",
			stack: &lokiv1.LokiStack{Status: lokiv1.LokiStackStatus{
				Storage: lokiv1.LokiStackStorageStatus{
					Schemas: []lokiv1.ObjectStorageSchemaStatus{
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV11,
								EffectiveDate: "2023-01-01",
							},
							EndDate: "2023-12-05",
							Status:  lokiv1.SchemaStatusInUse,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV12,
								EffectiveDate: "2023-12-03",
							},
							EndDate: "",
							Status:  lokiv1.SchemaStatusFuture,
						},
					},
				},
			}},
			now: testNow,
			wantUpgrade: &lokiv1.ProposedSchemaUpdate{
				UpgradeTime: testApplyTime,
				Schemas: []lokiv1.ObjectStorageSchema{
					{
						Version:       "v11",
						EffectiveDate: "2023-01-01",
					},
					{
						Version:       "v13",
						EffectiveDate: "2023-12-06",
					},
				},
			},
		},
		{
			desc: "upgrade - replace multiple future",
			stack: &lokiv1.LokiStack{Status: lokiv1.LokiStackStatus{
				Storage: lokiv1.LokiStackStorageStatus{
					Schemas: []lokiv1.ObjectStorageSchemaStatus{
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV11,
								EffectiveDate: "2023-01-01",
							},
							EndDate: "2023-12-05",
							Status:  lokiv1.SchemaStatusInUse,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV12,
								EffectiveDate: "2023-12-03",
							},
							EndDate: "2023-12-07",
							Status:  lokiv1.SchemaStatusFuture,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV13,
								EffectiveDate: "2023-12-08",
							},
							EndDate: "",
							Status:  lokiv1.SchemaStatusFuture,
						},
					},
				},
			}},
			now: testNow,
			wantUpgrade: &lokiv1.ProposedSchemaUpdate{
				UpgradeTime: testApplyTime,
				Schemas: []lokiv1.ObjectStorageSchema{
					{
						Version:       "v11",
						EffectiveDate: "2023-01-01",
					},
					{
						Version:       "v13",
						EffectiveDate: "2023-12-06",
					},
				},
			},
		},
		{
			desc: "combination: remove obsolete and add upgrade",
			stack: &lokiv1.LokiStack{Status: lokiv1.LokiStackStatus{
				Storage: lokiv1.LokiStackStorageStatus{
					Schemas: []lokiv1.ObjectStorageSchemaStatus{
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV11,
								EffectiveDate: "2022-12-01",
							},
							EndDate: "2022-12-31",
							Status:  lokiv1.SchemaStatusObsolete,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV12,
								EffectiveDate: "2023-01-01",
							},
							EndDate: "",
							Status:  lokiv1.SchemaStatusInUse,
						},
					},
				},
			}},
			now: testNow,
			wantUpgrade: &lokiv1.ProposedSchemaUpdate{
				UpgradeTime: testApplyTime,
				Schemas: []lokiv1.ObjectStorageSchema{
					{
						Version:       lokiv1.ObjectStorageSchemaV12,
						EffectiveDate: "2023-01-01",
					},
					{
						Version:       upgradeSchemaVersion,
						EffectiveDate: "2023-12-06",
					},
				},
			},
		},
		{
			desc: "combination: remove obsolete and replace multiple future",
			stack: &lokiv1.LokiStack{Status: lokiv1.LokiStackStatus{
				Storage: lokiv1.LokiStackStorageStatus{
					Schemas: []lokiv1.ObjectStorageSchemaStatus{
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       "v10",
								EffectiveDate: "2022-12-01",
							},
							EndDate: "2022-12-31",
							Status:  lokiv1.SchemaStatusObsolete,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV11,
								EffectiveDate: "2023-01-01",
							},
							EndDate: "2023-12-05",
							Status:  lokiv1.SchemaStatusInUse,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV12,
								EffectiveDate: "2023-12-03",
							},
							EndDate: "2023-12-07",
							Status:  lokiv1.SchemaStatusFuture,
						},
						{
							ObjectStorageSchema: lokiv1.ObjectStorageSchema{
								Version:       lokiv1.ObjectStorageSchemaV13,
								EffectiveDate: "2023-12-08",
							},
							EndDate: "",
							Status:  lokiv1.SchemaStatusFuture,
						},
					},
				},
			}},
			now: testNow,
			wantUpgrade: &lokiv1.ProposedSchemaUpdate{
				UpgradeTime: testApplyTime,
				Schemas: []lokiv1.ObjectStorageSchema{
					{
						Version:       "v11",
						EffectiveDate: "2023-01-01",
					},
					{
						Version:       "v13",
						EffectiveDate: "2023-12-06",
					},
				},
			},
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			upgrade, err := generateSchemaUpgrade(context.Background(), tc.stack, tc.now)
			if err != nil {
				t.Fatalf("got error: %s", err)
			}

			if diff := cmp.Diff(upgrade, tc.wantUpgrade); diff != "" {
				t.Errorf("upgrade differs: -got+want\n%s", diff)
			}
		})
	}
}
