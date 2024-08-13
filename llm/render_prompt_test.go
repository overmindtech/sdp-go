package llm

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/akedrou/textdiff"
	"github.com/akedrou/textdiff/myers"
	"github.com/google/uuid"
	"github.com/overmindtech/sdp-go"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTrim(t *testing.T) {
	t.Parallel()

	var trimmed string
	trimmed = trim(11, "1234567890")
	if trimmed != "1234567890" {
		t.Errorf("trim(10, \"1234567890\") = %#v; want 1234567890", trimmed)
	}

	trimmed = trim(10, "1234567890")
	if trimmed != "123\n…\n890" {
		t.Errorf("trim(10, \"1234567890\") = %#v; want 123\\n…\\n890", trimmed)
	}

	trimmed = trim(5, "1234567890")
	if trimmed != "12\n…\n90" {
		t.Errorf("trim(5, \"1234567890\") = %#v; want 12\\n…\\n90", trimmed)
	}
}

var testItemUUID = uuid.New()
var testFullyPopulatedItem = sdp.Item{
	Type:            "ec2-instance",
	UniqueAttribute: "id",
	Attributes: &sdp.ItemAttributes{
		AttrStruct: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"id": {
					Kind: &structpb.Value_StringValue{
						StringValue: "i-1234567890abcdef0",
					},
				},
				"ami-id": {
					Kind: &structpb.Value_StringValue{
						StringValue: "ami-12345678",
					},
				},
			},
		},
	},
	Metadata: &sdp.Metadata{
		SourceName: "ec2-instances",
		SourceQuery: &sdp.Query{
			Type:   "ec2-instance",
			Method: sdp.QueryMethod_GET,
			Query:  "i-1234567890abcdef0",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth:                  0,
				FollowOnlyBlastPropagation: true,
			},
			Scope:       "*",
			IgnoreCache: false,
			UUID:        testItemUUID[:],
			Deadline:    timestamppb.Now(),
		},
		Timestamp:             timestamppb.Now(),
		SourceDuration:        durationpb.New(5 * time.Second),
		SourceDurationPerItem: durationpb.New(1 * time.Second),
		Hidden:                false,
	},
	Scope: "6283467234.eu-west-1",
	LinkedItemQueries: []*sdp.LinkedItemQuery{
		{
			Query: &sdp.Query{
				Type:   "ec2-security-group",
				Method: sdp.QueryMethod_SEARCH,
				Query:  "sg-1234567890abcdef0",
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth:                  1,
					FollowOnlyBlastPropagation: false,
				},
				Scope:       "6283467234.eu-west-1",
				IgnoreCache: false,
				UUID:        testItemUUID[:],
				Deadline:    timestamppb.New(time.Now().Add(5 * time.Minute)),
			},
		},
	},
	LinkedItems: []*sdp.LinkedItem{
		{
			Item: &sdp.Reference{
				Type:                 "ebs-volume",
				UniqueAttributeValue: "v-1234567890abcdef0",
				Scope:                "6283467234.eu-west-1",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
	},
	Health: sdp.Health_HEALTH_OK.Enum(),
	Tags: map[string]string{
		"Name": "ExampleInstance",
	},
}

func TestItemToYAML(t *testing.T) {
	t.Run("with a good item", func(t *testing.T) {
		t.Parallel()

		yaml, err := itemToYAML(&testFullyPopulatedItem)

		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(yaml)
	})

	t.Run("with nil", func(t *testing.T) {
		t.Parallel()

		yaml, err := itemToYAML(nil)

		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(yaml)
	})

}

const testBefore = `    image: ghcr.io/overmindtech/api-server@sha256:bc92d3ff4feb5a1f29b1fcae67a32cdf99099b435890d0436bc4c85cd63a4e4c
image_pull_policy: IfNotPresent
name: api-server
port:
  - container_port: 8080
	host_port: 0
	protocol: TCP
readiness_probe:
  - failure_threshold: 3
	http_get:
	  - path: /healthz
		port: \"8080\"
		scheme: HTTP
	initial_delay_seconds: 0
	period_seconds: 1
	success_threshold: 1
	timeout_seconds: 1
resources:
  - limits:
	  memory: 2Gi
	requests:
	  cpu: 250m
	  memory: 200Mi
stdin: false
stdin_once: false
termination_message_path: /dev/termination-log
termination_message_policy: File
tty: false
volume_mount:
  - mount_path: /nats-nkeys
	mount_propagation: None
	name: nats-nkeys
	read_only: false
dns_policy: ClusterFirst
enable_service_links: true
host_ipc: false
host_network: false
host_pid: false
image_pull_secrets:
- name: srcman-registry-credentials
init_container:
- args:
  - --nsc-location
  - --nsc-operator
  - --write-secret
  - /nats-nkeys/nsc
  - api-server-generated-keys
  - dogfood
  - init
env_from:
  - secret_ref:
	  - name: o11y-secrets
		optional: false
image: ghcr.io/overmindtech/api-server@sha256:bc92d3ff4feb5a1f29b1fcae67a32cdf99099b435890d0436bc4c85cd63a4e4c`

const testAfter = `    image: ghcr.io/overmindtech/api-server@sha256:462be60d478b3e86796ce628765ec3f42b6b051b9c91a466ea70f47eebb64a09
image_pull_policy: IfNotPresent
name: api-server
port:
  - container_port: 8080
	host_port: 0
	protocol: TCP
readiness_probe:
  - failure_threshold: 3
	http_get:
	  - path: /healthz
		port: \"8080\"
		scheme: HTTP
	initial_delay_seconds: 0
	period_seconds: 1
	success_threshold: 1
	timeout_seconds: 1
resources:
  - limits:
	  memory: 2Gi
	requests:
	  cpu: 250m
	  memory: 200Mi
stdin: false
stdin_once: false
termination_message_path: /dev/termination-log
termination_message_policy: File
tty: false
volume_mount:
  - mount_path: /nats-nkeys
	mount_propagation: None
	name: nats-nkeys
	read_only: false
dns_policy: ClusterFirst
enable_service_links: true
host_ipc: false
host_network: false
host_pid: false
image_pull_secrets:
- name: srcman-registry-credentials
init_container:
- args:
  - --nsc-location
  - --nsc-operator
  - --write-secret
  - /nats-nkeys/nsc
  - api-server-generated-keys
  - dogfood
  - init
env_from:
  - secret_ref:
	  - name: o11y-secrets
		optional: false
image: ghcr.io/overmindtech/api-server@sha256:462be60d478b3e86796ce628765ec3f42b6b051b9c91a466ea70f47eebb64a09`

func TestDiffLibrary(t *testing.T) {
	edits := myers.ComputeEdits(testBefore, testAfter)
	unified, _ := textdiff.ToUnified("name", "name", testBefore, edits)

	fmt.Println(fmt.Sprint(unified))
}

func TestItemDiffToYAMLDiff(t *testing.T) {
	t.Run("with an actual difference", func(t *testing.T) {
		after := sdp.Item{}
		testFullyPopulatedItem.Copy(&after)
		after.GetAttributes().Set("ami-id", "ami-87654321")

		itemDiff := sdp.ItemDiff{
			Item:   testFullyPopulatedItem.Reference(),
			Status: sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED,
			Before: &testFullyPopulatedItem,
			After:  &after,
		}

		diff, err := itemDiffToYAMLDiff(&itemDiff)

		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(diff, "-    ami-id: ami-12345678") {
			t.Errorf("diff does not contain expected value")
		}
	})

}

func TestPrettyStatus(t *testing.T) {
	// Test all of the possible statuses
	tests := map[any]string{
		sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNSPECIFIED: "unspecified",
		sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNCHANGED:   "unchanged",
		sdp.ItemDiffStatus_ITEM_DIFF_STATUS_CREATED:     "created",
		sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED:     "updated",
		sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED:     "deleted",
		sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED:    "replaced",
		sdp.ChangeStatus_CHANGE_STATUS_UNSPECIFIED:      "unspecified",
		sdp.ChangeStatus_CHANGE_STATUS_DEFINING:         "defining",
		sdp.ChangeStatus_CHANGE_STATUS_HAPPENING:        "happening",
		sdp.ChangeStatus_CHANGE_STATUS_PROCESSING:       "processing",
		sdp.ChangeStatus_CHANGE_STATUS_DONE:             "done",
	}

	for status, expected := range tests {
		result := prettyStatus(status)
		if result != expected {
			t.Errorf("prettyStatus(%v) = %v; want %v", status, result, expected)
		}
	}
}
