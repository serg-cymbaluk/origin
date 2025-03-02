package synthetictests

import (
	"regexp"
	"strings"
	"testing"
	"time"

	v1 "github.com/openshift/api/config/v1"
	"github.com/openshift/origin/pkg/monitor/monitorapi"
)

func TestEventCountExtractor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
		times   int
	}{
		{
			name:    "simple",
			input:   `pod/network-check-target-5f44k node/ip-10-0-210-155.us-west-2.compute.internal - reason/NetworkNotReady network is not ready: container runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:Network plugin returns error: No CNI configuration file in /etc/kubernetes/cni/net.d/. Has your network provider started? (24 times)`,
			message: `pod/network-check-target-5f44k node/ip-10-0-210-155.us-west-2.compute.internal - reason/NetworkNotReady network is not ready: container runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:Network plugin returns error: No CNI configuration file in /etc/kubernetes/cni/net.d/. Has your network provider started?`,
			times:   24,
		},
		{
			name:    "new lines",
			input:   "ns/e2e-container-probe-7285 pod/liveness-f0fce2c6-6eed-4ace-bf69-2df5e5b8b1ea node/ci-op-sti304mj-2a78c-pq5zv-worker-b-sknbn reason/ProbeWarning Liveness probe warning: <a href=\"http://0.0.0.0/\">Found</a>.\n\n (22 times)",
			message: "ns/e2e-container-probe-7285 pod/liveness-f0fce2c6-6eed-4ace-bf69-2df5e5b8b1ea node/ci-op-sti304mj-2a78c-pq5zv-worker-b-sknbn reason/ProbeWarning Liveness probe warning: <a href=\"http://0.0.0.0/\">Found</a>.\n\n",
			times:   22,
		},
		{
			name:  "other message",
			input: "some node message",
			times: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualMessage, actualCount := getTimesAnEventHappened(test.input)
			if actualCount != test.times {
				t.Error(actualCount)
			}
			if actualMessage != test.message {
				t.Error(actualMessage)
			}
		})
	}
}

func TestEventRegexExcluder(t *testing.T) {
	allowedRepeatedEventsRegex := combinedRegexp(allowedRepeatedEventPatterns...)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "port-forward",
			message: `ns/e2e-port-forwarding-588 pod/pfpod node/ci-op-g1d5csj7-b08f5-fgrqd-worker-b-xj89f - reason/Unhealthy Readiness probe failed:`,
		},
		{
			name:    "container-probe",
			message: ` ns/e2e-container-probe-3794 pod/test-webserver-3faa80d6-05f2-42a7-9846-099e8a4cf28c node/ci-op-gzm3mjwm-875d2-tvchv-worker-c-w47mw - reason/Unhealthy Readiness probe failed: Get "http://10.131.0.54:81/": dial tcp 10.131.0.54:81: connect: connection refused`,
		},
		{
			name:    "failing-init-container",
			message: `ns/e2e-init-container-368 pod/pod-init-cb40ee55-e9c5-4c4b-b541-47cc018d9856 node/ci-op-ncxkp5gj-875d2-5jcfn-worker-c-pwf97 - reason/BackOff Back-off restarting failed container`,
		},
		{
			name:    "scc-test-3",
			message: `ns/e2e-test-scc-578l5 pod/test3 - reason/FailedScheduling 0/6 nodes are available: 3 node(s) didn't match Pod's node affinity/selector, 3 node(s) had taint {node-role.kubernetes.io/master: }, that the pod didn't tolerate.`,
		},
		{
			name:    "missing image",
			message: `ns/e2e-deployment-478 pod/webserver-deployment-795d758f88-fdr4d node/ci-op-h1wxg6l0-16f7c-mb4sj-worker-b-wcdcf - reason/BackOff Back-off pulling image "webserver:404"`,
		},
		{
			name:    "non-root",
			message: `ns/e2e-security-context-test-6596 pod/explicit-root-uid node/ci-op-isj7rd3k-2a78c-kk69w-worker-a-v4kdb - reason/Failed Error: container's runAsUser breaks non-root policy (pod: "explicit-root-uid_e2e-security-context-test-6596(22bf29d0-e546-4a15-8dd7-8acd9165c924)", container: explicit-root-uid)`,
		},
		{
			name:    "local-volume-failed-scheduling",
			message: `ns/e2e-persistent-local-volumes-test-7012 pod/pod-940713ce-7645-4d8c-bba0-5705350a5655 reason/FailedScheduling 0/6 nodes are available: 1 node(s) had volume node affinity conflict, 2 node(s) didn't match Pod's node affinity/selector, 3 node(s) had taint {node-role.kubernetes.io/master: }, that the pod didn't tolerate. (2 times)`,
		},
		{
			name:    "vsphere-hw-13-default-upi-install",
			message: `ns/openshift-cluster-storage-operator deployment/vsphere-problem-detector-operator - reason/VSphereOlderVersionDetected Marking cluster un-upgradeable because one or more VMs are on hardware version vmx-13`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := allowedRepeatedEventsRegex.MatchString(test.message)
			if !actual {
				t.Fatal("did not match")
			}
		})
	}

}

func TestUpgradeEventRegexExcluder(t *testing.T) {
	allowedRepeatedEventsRegex := combinedRegexp(allowedUpgradeRepeatedEventPatterns...)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "etcd-member",
			message: `ns/openshift-etcd-operator deployment/etcd-operator - reason/UnhealthyEtcdMember unhealthy members: ip-10-0-198-128.ec2.internal`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := allowedRepeatedEventsRegex.MatchString(test.message)
			if !actual {
				t.Fatal("did not match")
			}
		})
	}

}

func TestKnownBugEvents(t *testing.T) {
	evaluator := duplicateEventsEvaluator{
		allowedRepeatedEventPatterns: allowedRepeatedEventPatterns,
		knownRepeatedEventsBugs: []knownProblem{
			{
				Regexp: regexp.MustCompile(`ns/.* reason/SomeEvent1.*`),
				BZ:     "https://bugzilla.redhat.com/show_bug.cgi?id=1234567",
			},
			{
				Regexp:   regexp.MustCompile("ns/.*reason/SomeEvent2.*"),
				BZ:       "https://bugzilla.redhat.com/show_bug.cgi?id=1234567",
				Topology: topologyPointer(v1.SingleReplicaTopologyMode),
			},
			{
				Regexp:   regexp.MustCompile("ns/.*reason/SomeEvent3.*"),
				BZ:       "https://bugzilla.redhat.com/show_bug.cgi?id=1234567",
				Platform: platformPointer(v1.AWSPlatformType),
			},
			{
				Regexp:   regexp.MustCompile("ns/.*reason/SomeEvent4.*"),
				BZ:       "https://bugzilla.redhat.com/show_bug.cgi?id=1234567",
				Topology: topologyPointer(v1.HighlyAvailableTopologyMode),
			},
			{
				Regexp:   regexp.MustCompile("ns/.*reason/SomeEvent5.*"),
				BZ:       "https://bugzilla.redhat.com/show_bug.cgi?id=1234567",
				Platform: platformPointer(v1.GCPPlatformType),
			},
			{
				Regexp:   regexp.MustCompile("ns/.*reason/SomeEvent6.*"),
				BZ:       "https://bugzilla.redhat.com/show_bug.cgi?id=1234567",
				Platform: platformPointer(""),
			},
		},
	}

	tests := []struct {
		name     string
		message  string
		match    bool
		platform v1.PlatformType
		topology v1.TopologyMode
	}{
		{
			name:     "matches without platform or topology",
			message:  `ns/e2e - reason/SomeEvent1 foo (21 times)`,
			match:    true,
			platform: v1.AWSPlatformType,
			topology: v1.SingleReplicaTopologyMode},
		{
			name:     "matches with topology",
			message:  `ns/e2e - reason/SomeEvent2 foo (21 times)`,
			match:    true,
			platform: v1.AWSPlatformType,
			topology: v1.SingleReplicaTopologyMode,
		},
		{
			name:     "matches with topology and platform",
			message:  `ns/e2e - reason/SomeEvent3 foo (21 times)`,
			match:    true,
			platform: v1.AWSPlatformType,
			topology: v1.SingleReplicaTopologyMode,
		},
		{
			name:     "does not match against different topology",
			message:  `ns/e2e - reason/SomeEvent4 foo (21 times)`,
			platform: v1.AWSPlatformType,
			topology: v1.SingleReplicaTopologyMode,
			match:    false,
		},
		{
			name:     "does not match against different platform",
			message:  `ns/e2e - reason/SomeEvent5 foo (21 times)`,
			platform: v1.AWSPlatformType,
			topology: v1.SingleReplicaTopologyMode,
			match:    false,
		},
		{
			name:     "empty platform matches empty platform",
			message:  `ns/e2e - reason/SomeEvent6 foo (21 times)`,
			platform: "",
			match:    true,
		},
		{
			name:     "empty platform doesn't match another platform",
			message:  `ns/e2e - reason/SomeEvent6 foo (21 times)`,
			platform: v1.AWSPlatformType,
			match:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := monitorapi.Intervals{}
			events = append(events,
				monitorapi.EventInterval{
					Condition: monitorapi.Condition{Message: test.message},
					From:      time.Unix(1, 0),
					To:        time.Unix(1, 0)},
			)
			evaluator.platform = test.platform
			evaluator.topology = test.topology

			junits := evaluator.testDuplicatedEvents("events should not repeat", false, events, nil)
			if len(junits) < 1 {
				t.Fatal("didn't get junit for duplicated event")
			}
			if test.match && !strings.Contains(junits[0].FailureOutput.Output, "1 events with known BZs") {
				t.Fatalf("expected case to match, but it didn't: %s", test.name)
			}

			if !test.match && strings.Contains(junits[0].FailureOutput.Output, "1 events with known BZs") {
				t.Fatalf("expected case to not match, but it did: %s", test.name)
			}
		})
	}
}
