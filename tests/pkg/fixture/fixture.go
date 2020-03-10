// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package fixture

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/tests"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var (
	BestEffort    = corev1.ResourceRequirements{}
	BurstbleSmall = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}
	BurstbleMedium = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2000m"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}

	clusterCA = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUR1akNDQXFLZ0F3SUJBZ0lVWnA1TlFycDhJem9jRGJhWnM5aEtsYjlwc3lVd0RRWUpLb1pJaHZjTkFRRUwKQlFBd1l6RUxNQWtHQTFVRUJoTUNWVk14RURBT0JnTlZCQWdUQjBKbGFXcHBibWN4Q3pBSkJnTlZCQWNUQWtOQgpNUkF3RGdZRFZRUUtFd2RRYVc1blEwRlFNUTB3Q3dZRFZRUUxFd1JVYVVSQ01SUXdFZ1lEVlFRREV3dFVhVVJDCklGTmxjblpsY2pBZUZ3MHlNREF6TVRBd016VXhNREJhRncweU5UQXpNRGt3TXpVeE1EQmFNR014Q3pBSkJnTlYKQkFZVEFsVlRNUkF3RGdZRFZRUUlFd2RDWldscWFXNW5NUXN3Q1FZRFZRUUhFd0pEUVRFUU1BNEdBMVVFQ2hNSApVR2x1WjBOQlVERU5NQXNHQTFVRUN4TUVWR2xFUWpFVU1CSUdBMVVFQXhNTFZHbEVRaUJUWlhKMlpYSXdnZ0VpCk1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLQW9JQkFRRGJQNnVXN2llNTNvSGtnVU1IajB5eTdaREQKSll0SEJTYTJ0SUFsUU1zdmlPTUU0Q1hVeUN6ekx6WHJqVzBQOFZiMmJvSmVRcmplOVUxOElqRG9wL0dEaFVGdgpQSllmNStuZm1mMS9MemRoZ3NvMU8yeFhsYnppS1JwbllPNUQ3T3B2YkFJRndGek1xQmtud1hWbFgvdlZlQU9vCkRQK254OWZnbWpNbWQ1TDg1K0JSMjVSdFBoZEY4Ylk1V203V0o4UWdGbUE4LzVjOTE0MkV1emVXVnJUaFM0T1AKK0ZjeUdWY1BVQllzbjBLc2hodjRGa3FjV1l5TVNlSGp0cG5YczFjM2JzakZRcjRobWsrZVZoV0JRaDVlT01pbgorMHVBQjhuZWRlSFlGR29Gbm5LbWlPbm9XWXZrNDR4RlprTTdyaXMyNGhQNW9mZ3dDYXlucFpRK0RSSXZBZ01CCkFBR2paakJrTUE0R0ExVWREd0VCL3dRRUF3SUJCakFTQmdOVkhSTUJBZjhFQ0RBR0FRSC9BZ0VDTUIwR0ExVWQKRGdRV0JCU3B6L2Zld0IzVlBvSUNYZGJZOVFrTjlGOE5hREFmQmdOVkhTTUVHREFXZ0JTcHovZmV3QjNWUG9JQwpYZGJZOVFrTjlGOE5hREFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBUXJ4STlXbElLTS9pYmJCcWdkVmwySmozCjVBR2JwaVUrem5kMGMwdi9JS3VzS21YbG1UdTdzcW1kTWVJeWFXSjJHclBBRlMvbkJpSnA3WVJ6Z2VTQWdqOGIKMWhXS2VoRlM3NkFYdjNRbFZ1RkZyUWViUHBLVmp2R2VxZ3dlRW5jS2ZqdldTendwS3h1dW5xQk1OdVdETmtlcQpEUi9CQlJNblltelVPYTRQclU3THQ0Q0F3b1NIc0dCanZzMEZxNVcxUjhuQVllNUxHd01seGFGbmZ1Lzg3a3g3Cnp6eUo0QnBhcldpeGY1QjR0SUZEdEJjRjhWTjk3ZWRJeDc3U21Cb3Blb2xXVjIreVQ2RTRuWm55WnQ3aGhSZGMKQ0NNblZlOW5tZVEwZHFJaXEveDQ5dDk5MzN2aHFpbVYvbzJ5dzRWVCt2alRaTVRGUkJvcnRWVHRsUlBZZ0E9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	clusterPDCrt = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZiVENDQkZXZ0F3SUJBZ0lVTU9vZkpQZUQ0SFNoR3pVNEpIQ2NxKzFrNUJBd0RRWUpLb1pJaHZjTkFRRUwKQlFBd1l6RUxNQWtHQTFVRUJoTUNWVk14RURBT0JnTlZCQWdUQjBKbGFXcHBibWN4Q3pBSkJnTlZCQWNUQWtOQgpNUkF3RGdZRFZRUUtFd2RRYVc1blEwRlFNUTB3Q3dZRFZRUUxFd1JVYVVSQ01SUXdFZ1lEVlFRREV3dFVhVVJDCklGTmxjblpsY2pBZUZ3MHlNREF6TVRBd016VXpNREJhRncweU5UQXpNRGt3TXpVek1EQmFNRWt4Q3pBSkJnTlYKQkFZVEFsVlRNUll3RkFZRFZRUUlFdzFUWVc0Z1JuSmhibU5wYzJOdk1Rc3dDUVlEVlFRSEV3SkRRVEVWTUJNRwpBMVVFQXhNTVZHbEVRaUJEYkhWemRHVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUU0VXBGCmtuZWExRWkwa2NNL3FpdW5QZURwSHpKS2toVWtBVnB6eVVoTzI2Si9rWXdPTURiQW15WnhjNXdDRk54cUplMDcKZitvNnZsVDVYOUx2VmYrSDZxT0NBdnd3Z2dMNE1BNEdBMVVkRHdFQi93UUVBd0lGb0RBZEJnTlZIU1VFRmpBVQpCZ2dyQmdFRkJRY0RBUVlJS3dZQkJRVUhBd0l3REFZRFZSMFRBUUgvQkFJd0FEQWRCZ05WSFE0RUZnUVU1N0RVClNaSlVMYTc4RWlzN2llcHNOVkk3cGpJd0h3WURWUjBqQkJnd0ZvQVVxYy8zM3NBZDFUNkNBbDNXMlBVSkRmUmYKRFdnd2dnSjNCZ05WSFJFRWdnSnVNSUlDYW9JSVltRnphV010Y0dTQ0VHSmhjMmxqTFhCa0xtUmxabUYxYkhTQwpGR0poYzJsakxYQmtMbVJsWm1GMWJIUXVjM1pqZ2cxaVlYTnBZeTF3WkMxd1pXVnlnaFZpWVhOcFl5MXdaQzF3ClpXVnlMbVJsWm1GMWJIU0NHV0poYzJsakxYQmtMWEJsWlhJdVpHVm1ZWFZzZEM1emRtT0NEeW91WW1GemFXTXQKY0dRdGNHVmxjb0lYS2k1aVlYTnBZeTF3WkMxd1pXVnlMbVJsWm1GMWJIU0NHeW91WW1GemFXTXRjR1F0Y0dWbApjaTVrWldaaGRXeDBMbk4yWTRJS1ltRnphV010ZEdscmRvSVNZbUZ6YVdNdGRHbHJkaTVrWldaaGRXeDBnaFppCllYTnBZeTEwYVd0MkxtUmxabUYxYkhRdWMzWmpnZzlpWVhOcFl5MTBhV3QyTFhCbFpYS0NGMkpoYzJsakxYUnAKYTNZdGNHVmxjaTVrWldaaGRXeDBnaHRpWVhOcFl5MTBhV3QyTFhCbFpYSXVaR1ZtWVhWc2RDNXpkbU9DRVNvdQpZbUZ6YVdNdGRHbHJkaTF3WldWeWdoa3FMbUpoYzJsakxYUnBhM1l0Y0dWbGNpNWtaV1poZFd4MGdoMHFMbUpoCmMybGpMWFJwYTNZdGNHVmxjaTVrWldaaGRXeDBMbk4yWTRJS1ltRnphV010ZEdsa1lvSVNZbUZ6YVdNdGRHbGsKWWk1a1pXWmhkV3gwZ2haaVlYTnBZeTEwYVdSaUxtUmxabUYxYkhRdWMzWmpnZzlpWVhOcFl5MTBhV1JpTFhCbApaWEtDRjJKaGMybGpMWFJwWkdJdGNHVmxjaTVrWldaaGRXeDBnaHRpWVhOcFl5MTBhV1JpTFhCbFpYSXVaR1ZtCllYVnNkQzV6ZG1PQ0VTb3VZbUZ6YVdNdGRHbGtZaTF3WldWeWdoa3FMbUpoYzJsakxYUnBaR0l0Y0dWbGNpNWsKWldaaGRXeDBnaDBxTG1KaGMybGpMWFJwWkdJdGNHVmxjaTVrWldaaGRXeDBMbk4yWTRjRWZ3QUFBWWNRQUFBQQpBQUFBQUFBQUFBQUFBQUFBQVRBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQUU2bDcrenEveFFoaEpOdnVzWGpMCmNIUUlUSVFYQm5vZ3hCcXdCY1JMMFYwaHBDTGVnTVdtWm5aSEVXZDZMb2svOTMzZkxWM2pZcktlSUNiMFJhZkQKRWdKdGI5cFZjNUtoYzNmQlptcWxjZW9wa2xhdDl3KzMzT1g5eFBPRnVaWjJCdEI3clJ1VUM3RHZEN3F6RUdlTQo3emZlVVQxQzhuUW12N21HMFdWMUxSS0xjSDlOZFJtZlhUakYvbzdXUGtRbWpxYU1qTWJKeTJRY2dTem9iTEYrCnpTU3habnFGMVNMTjJvZzc4cXVqYU55dUNaZFRsL1ltNHQzeDlMZjhKSGhsY2dEYWREK2xTY1hjbFdTOU1YZTYKVDVSVjVkOW1NUnBEUXZRazcxeTY5SjRkR0xzMmp6UTZHaXBEZjhJeTZQVGpiMWp0bXhQM2VRQWFpc0NZYlU0cAptZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
	clusterPDKey = "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUYrWi9YRnVmZ3RQVld2cnQvMFNpR09JdnJEVFFEcllDQ2gyQUx2UFNONzFvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFNFVwRmtuZWExRWkwa2NNL3FpdW5QZURwSHpKS2toVWtBVnB6eVVoTzI2Si9rWXdPTURiQQpteVp4YzV3Q0ZOeHFKZTA3ZitvNnZsVDVYOUx2VmYrSDZnPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="
	clusterTiKVCrt = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZiVENDQkZXZ0F3SUJBZ0lVQXI1bzV5L0RaMDBML0FndXl2eVNLV0RRN0c0d0RRWUpLb1pJaHZjTkFRRUwKQlFBd1l6RUxNQWtHQTFVRUJoTUNWVk14RURBT0JnTlZCQWdUQjBKbGFXcHBibWN4Q3pBSkJnTlZCQWNUQWtOQgpNUkF3RGdZRFZRUUtFd2RRYVc1blEwRlFNUTB3Q3dZRFZRUUxFd1JVYVVSQ01SUXdFZ1lEVlFRREV3dFVhVVJDCklGTmxjblpsY2pBZUZ3MHlNREF6TVRBd016VXpNREJhRncweU5UQXpNRGt3TXpVek1EQmFNRWt4Q3pBSkJnTlYKQkFZVEFsVlRNUll3RkFZRFZRUUlFdzFUWVc0Z1JuSmhibU5wYzJOdk1Rc3dDUVlEVlFRSEV3SkRRVEVWTUJNRwpBMVVFQXhNTVZHbEVRaUJEYkhWemRHVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVwMnlhCnU1SkEyNEpELzgwSVVjTnFwNnVNbE1aRktlWWVNNDRFU3ZmdmFFTGdBNjdGcTl2U0gyblBGd3luVXpPd3M5UWQKOXIwZ3VYdE1zY1puK1pFS0hxT0NBdnd3Z2dMNE1BNEdBMVVkRHdFQi93UUVBd0lGb0RBZEJnTlZIU1VFRmpBVQpCZ2dyQmdFRkJRY0RBUVlJS3dZQkJRVUhBd0l3REFZRFZSMFRBUUgvQkFJd0FEQWRCZ05WSFE0RUZnUVVkak10Cm83TWQvWFRaQVJPbFV3TmRSYjZvUlBzd0h3WURWUjBqQkJnd0ZvQVVxYy8zM3NBZDFUNkNBbDNXMlBVSkRmUmYKRFdnd2dnSjNCZ05WSFJFRWdnSnVNSUlDYW9JSVltRnphV010Y0dTQ0VHSmhjMmxqTFhCa0xtUmxabUYxYkhTQwpGR0poYzJsakxYQmtMbVJsWm1GMWJIUXVjM1pqZ2cxaVlYTnBZeTF3WkMxd1pXVnlnaFZpWVhOcFl5MXdaQzF3ClpXVnlMbVJsWm1GMWJIU0NHV0poYzJsakxYQmtMWEJsWlhJdVpHVm1ZWFZzZEM1emRtT0NEeW91WW1GemFXTXQKY0dRdGNHVmxjb0lYS2k1aVlYTnBZeTF3WkMxd1pXVnlMbVJsWm1GMWJIU0NHeW91WW1GemFXTXRjR1F0Y0dWbApjaTVrWldaaGRXeDBMbk4yWTRJS1ltRnphV010ZEdscmRvSVNZbUZ6YVdNdGRHbHJkaTVrWldaaGRXeDBnaFppCllYTnBZeTEwYVd0MkxtUmxabUYxYkhRdWMzWmpnZzlpWVhOcFl5MTBhV3QyTFhCbFpYS0NGMkpoYzJsakxYUnAKYTNZdGNHVmxjaTVrWldaaGRXeDBnaHRpWVhOcFl5MTBhV3QyTFhCbFpYSXVaR1ZtWVhWc2RDNXpkbU9DRVNvdQpZbUZ6YVdNdGRHbHJkaTF3WldWeWdoa3FMbUpoYzJsakxYUnBhM1l0Y0dWbGNpNWtaV1poZFd4MGdoMHFMbUpoCmMybGpMWFJwYTNZdGNHVmxjaTVrWldaaGRXeDBMbk4yWTRJS1ltRnphV010ZEdsa1lvSVNZbUZ6YVdNdGRHbGsKWWk1a1pXWmhkV3gwZ2haaVlYTnBZeTEwYVdSaUxtUmxabUYxYkhRdWMzWmpnZzlpWVhOcFl5MTBhV1JpTFhCbApaWEtDRjJKaGMybGpMWFJwWkdJdGNHVmxjaTVrWldaaGRXeDBnaHRpWVhOcFl5MTBhV1JpTFhCbFpYSXVaR1ZtCllYVnNkQzV6ZG1PQ0VTb3VZbUZ6YVdNdGRHbGtZaTF3WldWeWdoa3FMbUpoYzJsakxYUnBaR0l0Y0dWbGNpNWsKWldaaGRXeDBnaDBxTG1KaGMybGpMWFJwWkdJdGNHVmxjaTVrWldaaGRXeDBMbk4yWTRjRWZ3QUFBWWNRQUFBQQpBQUFBQUFBQUFBQUFBQUFBQVRBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQVV5ekwrNGFLNi93UEZCTllXZXNEClhYSmdSYys1dCttaElQaWtYbG1DWjU1UUYrR2FpM0ROS1NRNU9ybnpVNTJ3V0svRXVRWE5nNThvZ1ZuNFBPZVoKSCtKQ0oveEJhOWFtdU14cUxtSnZRTUJMTzJYSkFULzFrY2NEdkMvZDl0LzZyVmxmT0crY1g5NkFUT2cvR25WQQpyRjBSbVRxL1hqUVZkRlJ3VHB3Rk5jOXpiR0d5VkkvUGtPQ0RKUGc2OTlBSVV3S3hNbzVYa3gxeEZaU3ZxczNnCkhlUU1QQnRiRFNmTFpzT3RGNElvczRtRzVFTnNvdU1wdngvSkdTVlJkV1NRajNvWnVUWHB6L2htKzJRN1M3azgKcFJYUlFOdjV6VFgySVRldUNpTWF0OU9BY2VOODRtRDBZMjhaRXlWdWR2WWUzVUdYV2FHbExKNFlTZnhSWUZsVgorZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
	clusterTiKVKey = "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUxGME4yMm5oZ1FoNzZqbDhadmN3VWVkNzVrQlIzY2pnM0ZZdy9GMFVtNkpvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFcDJ5YXU1SkEyNEpELzgwSVVjTnFwNnVNbE1aRktlWWVNNDRFU3ZmdmFFTGdBNjdGcTl2UwpIMm5QRnd5blV6T3dzOVFkOXIwZ3VYdE1zY1puK1pFS0hnPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="
	clusterTiDBCrt = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZiVENDQkZXZ0F3SUJBZ0lVSGVTQ05aMCtiRHRPV1RIaEU4YkJPRDZwV1drd0RRWUpLb1pJaHZjTkFRRUwKQlFBd1l6RUxNQWtHQTFVRUJoTUNWVk14RURBT0JnTlZCQWdUQjBKbGFXcHBibWN4Q3pBSkJnTlZCQWNUQWtOQgpNUkF3RGdZRFZRUUtFd2RRYVc1blEwRlFNUTB3Q3dZRFZRUUxFd1JVYVVSQ01SUXdFZ1lEVlFRREV3dFVhVVJDCklGTmxjblpsY2pBZUZ3MHlNREF6TVRBd016VXpNREJhRncweU5UQXpNRGt3TXpVek1EQmFNRWt4Q3pBSkJnTlYKQkFZVEFsVlRNUll3RkFZRFZRUUlFdzFUWVc0Z1JuSmhibU5wYzJOdk1Rc3dDUVlEVlFRSEV3SkRRVEVWTUJNRwpBMVVFQXhNTVZHbEVRaUJEYkhWemRHVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUV4WlJhCkRKN2doTnRBU0NwdmFPM2p5bHBpdXI2UnA0UXRBZzNWczRrMEY2SEZsa2I2SWg1THlYT0VENzFkcCtUREgxVVQKWnhnQ2dzYllLUW0wdjFDQ29xT0NBdnd3Z2dMNE1BNEdBMVVkRHdFQi93UUVBd0lGb0RBZEJnTlZIU1VFRmpBVQpCZ2dyQmdFRkJRY0RBUVlJS3dZQkJRVUhBd0l3REFZRFZSMFRBUUgvQkFJd0FEQWRCZ05WSFE0RUZnUVVVaEQzCjUrWmNiNlA2K0xiWVBqRGdiRlVmZ280d0h3WURWUjBqQkJnd0ZvQVVxYy8zM3NBZDFUNkNBbDNXMlBVSkRmUmYKRFdnd2dnSjNCZ05WSFJFRWdnSnVNSUlDYW9JSVltRnphV010Y0dTQ0VHSmhjMmxqTFhCa0xtUmxabUYxYkhTQwpGR0poYzJsakxYQmtMbVJsWm1GMWJIUXVjM1pqZ2cxaVlYTnBZeTF3WkMxd1pXVnlnaFZpWVhOcFl5MXdaQzF3ClpXVnlMbVJsWm1GMWJIU0NHV0poYzJsakxYQmtMWEJsWlhJdVpHVm1ZWFZzZEM1emRtT0NEeW91WW1GemFXTXQKY0dRdGNHVmxjb0lYS2k1aVlYTnBZeTF3WkMxd1pXVnlMbVJsWm1GMWJIU0NHeW91WW1GemFXTXRjR1F0Y0dWbApjaTVrWldaaGRXeDBMbk4yWTRJS1ltRnphV010ZEdscmRvSVNZbUZ6YVdNdGRHbHJkaTVrWldaaGRXeDBnaFppCllYTnBZeTEwYVd0MkxtUmxabUYxYkhRdWMzWmpnZzlpWVhOcFl5MTBhV3QyTFhCbFpYS0NGMkpoYzJsakxYUnAKYTNZdGNHVmxjaTVrWldaaGRXeDBnaHRpWVhOcFl5MTBhV3QyTFhCbFpYSXVaR1ZtWVhWc2RDNXpkbU9DRVNvdQpZbUZ6YVdNdGRHbHJkaTF3WldWeWdoa3FMbUpoYzJsakxYUnBhM1l0Y0dWbGNpNWtaV1poZFd4MGdoMHFMbUpoCmMybGpMWFJwYTNZdGNHVmxjaTVrWldaaGRXeDBMbk4yWTRJS1ltRnphV010ZEdsa1lvSVNZbUZ6YVdNdGRHbGsKWWk1a1pXWmhkV3gwZ2haaVlYTnBZeTEwYVdSaUxtUmxabUYxYkhRdWMzWmpnZzlpWVhOcFl5MTBhV1JpTFhCbApaWEtDRjJKaGMybGpMWFJwWkdJdGNHVmxjaTVrWldaaGRXeDBnaHRpWVhOcFl5MTBhV1JpTFhCbFpYSXVaR1ZtCllYVnNkQzV6ZG1PQ0VTb3VZbUZ6YVdNdGRHbGtZaTF3WldWeWdoa3FMbUpoYzJsakxYUnBaR0l0Y0dWbGNpNWsKWldaaGRXeDBnaDBxTG1KaGMybGpMWFJwWkdJdGNHVmxjaTVrWldaaGRXeDBMbk4yWTRjRWZ3QUFBWWNRQUFBQQpBQUFBQUFBQUFBQUFBQUFBQVRBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQUZlUG52bldZRlBPdUNLN0tyb3BRCkdyYkdnMjVRRFV2TmFLaXIwY1ZwTHZqMWFwTDE2UmNHdG1rMi8rZGVnck9tanNnODNvVlp1Q3k3ejU4VkdiVTEKQVYvTWlHRjdoUkxCMEYwdldxdVRMdkVmZXVaZmRob0VySEZyK0hHSmRsY3h4cytSY2xCVE1JVGFLOTZ3S3FhVQp6WHhGRU1MWGdwZzVVaDI0V2pwZXBia0FIY0J5Z2piWThSYW54cVQyaUxmZHNNelQ1aGR2M1poVVRmbWg2QTM3CjQvWTJxcUtqeDNLdDJXTE5haHordGZnN1R0Z1lPUDZWVDM0YkQ5dW5DQmsrcGlueW5jcXVESS9JanZXRnovRUQKdTlyOFF3dkt4VVErOUVzVVBOU1ltQjJJaS9xOGluS1QrTWl4b0F6WUJmcnJ5Mjg4alR2bFNrVTdPaEp6aXUxNQpBQT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
	clusterTiDBKey = "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUpBMzdxSVBRcmtRTysrejdGUUlyUysxWnV5d0dVMDVlOHV3Wm5OYzdBMDJvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFeFpSYURKN2doTnRBU0NwdmFPM2p5bHBpdXI2UnA0UXRBZzNWczRrMEY2SEZsa2I2SWg1TAp5WE9FRDcxZHArVERIMVVUWnhnQ2dzYllLUW0wdjFDQ29nPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="
	clusterClientCrt = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM0ekNDQWN1Z0F3SUJBZ0lVU3hXaGpSbEJxbm9RdEVmdERCamRUVHJ3RnZRd0RRWUpLb1pJaHZjTkFRRUwKQlFBd1l6RUxNQWtHQTFVRUJoTUNWVk14RURBT0JnTlZCQWdUQjBKbGFXcHBibWN4Q3pBSkJnTlZCQWNUQWtOQgpNUkF3RGdZRFZRUUtFd2RRYVc1blEwRlFNUTB3Q3dZRFZRUUxFd1JVYVVSQ01SUXdFZ1lEVlFRREV3dFVhVVJDCklGTmxjblpsY2pBZUZ3MHlNREF6TVRBd016VTBNREJhRncweU5UQXpNRGt3TXpVME1EQmFNRWd4Q3pBSkJnTlYKQkFZVEFsVlRNUll3RkFZRFZRUUlFdzFUWVc0Z1JuSmhibU5wYzJOdk1Rc3dDUVlEVlFRSEV3SkRRVEVVTUJJRwpBMVVFQXhNTFpYaGhiWEJzWlM1dVpYUXdXVEFUQmdjcWhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBVDFraUlyCmJINVM5UW1pcHJvaEhKbTdibWZIdDFCWW8wOENFQTdOY2Q0VTlZcWd3VzNRQ0ozNHJwalJFaHo2STF5dTR5dFIKaFY5WWIwbnlpN1FlK3JYM28zVXdjekFPQmdOVkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQgpCUVVIQXdJd0RBWURWUjBUQVFIL0JBSXdBREFkQmdOVkhRNEVGZ1FVelpURWNRNFJwcFRrTkhWMitHUHBiUStyCjJRUXdId1lEVlIwakJCZ3dGb0FVcWMvMzNzQWQxVDZDQWwzVzJQVUpEZlJmRFdnd0RRWUpLb1pJaHZjTkFRRUwKQlFBRGdnRUJBS1ovN1U1YW1RT3RndHNEbHBCa001QU5hZGNkalNGNzQyWUtEUWNuSmJjemR4TndnclZxb1FZNQpvMzh3NmRMdG9KWkpEaTNoMFl4cEZ0RGcveTNNU29OZVhpNW0xald0Sms3bGExYkxJS0V0ODBJcTZJeXFSU2Z0CmNzTldUbURaRmpPeGVZcjlxV0U4U0JxQW9aZkx4cU5UK3dJTnAyTWRMSGw4eVZ3bEhzSHN2MWRibldEalRCa20KdTkrYUthZDJOSkx1dE9YYkx3Q0pJbFFmWnZjYkY3eWp4R2VXaW1rZ0ZobHc5T3ZJUmVyVFZya3phbUVOTWxYNwprRGUzZTNZY3RURDlWNnQvcnJZQmdPTTJ4WFJNZ01ZbFlxTFBUeDJUTXBIQ0pqVnJkeSszOE1Nd0VSdmYzU1A0ClpjdDBja0dobThUQXI2eXFDa0hIMXNNbm80NVNTUGM9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
	clusterClientKey = "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUhFaFVZNEdIWFZ4R2Z0c3VYODVpblF4c3lRdnNIY0JVSDkxa0lESjJJeTJvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFOVpJaUsyeCtVdlVKb3FhNklSeVp1MjVueDdkUVdLTlBBaEFPelhIZUZQV0tvTUZ0MEFpZAorSzZZMFJJYytpTmNydU1yVVlWZldHOUo4b3UwSHZxMTl3PT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="
)

func WithStorage(r corev1.ResourceRequirements, size string) corev1.ResourceRequirements {
	if r.Requests == nil {
		r.Requests = corev1.ResourceList{}
	}
	r.Requests[corev1.ResourceStorage] = resource.MustParse(size)

	return r
}

// GetTidbCluster returns a TidbCluster resource configured for testing
func GetTidbCluster(ns, name, version string) *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbClusterSpec{
			Version:         version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			PVReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			SchedulerName:   "tidb-scheduler",
			Timezone:        "Asia/Shanghai",

			PD: v1alpha1.PDSpec{
				Replicas:             3,
				BaseImage:            "pingcap/pd",
				ResourceRequirements: WithStorage(BurstbleSmall, "1Gi"),
				Config: &v1alpha1.PDConfig{
					Log: &v1alpha1.PDLogConfig{
						Level: "info",
					},
					// accelerate failover
					Schedule: &v1alpha1.PDScheduleConfig{
						MaxStoreDownTime: "5m",
					},
				},
			},

			TiKV: v1alpha1.TiKVSpec{
				Replicas:             3,
				BaseImage:            "pingcap/tikv",
				ResourceRequirements: WithStorage(BurstbleMedium, "10Gi"),
				MaxFailoverCount:     pointer.Int32Ptr(3),
				Config: &v1alpha1.TiKVConfig{
					LogLevel: "info",
					Server:   &v1alpha1.TiKVServerConfig{},
				},
			},

			TiDB: v1alpha1.TiDBSpec{
				Replicas:             2,
				BaseImage:            "pingcap/tidb",
				ResourceRequirements: BurstbleMedium,
				Service: &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
					ExposeStatus: pointer.BoolPtr(true),
				},
				SeparateSlowLog:  pointer.BoolPtr(true),
				MaxFailoverCount: pointer.Int32Ptr(3),
				Config: &v1alpha1.TiDBConfig{
					Log: &v1alpha1.Log{
						Level: pointer.StringPtr("info"),
					},
				},
			},
		},
	}
}

func NewTidbMonitor(name, namespace string, tc *v1alpha1.TidbCluster, grafanaEnabled, persist bool) *v1alpha1.TidbMonitor {
	imagePullPolicy := corev1.PullIfNotPresent
	monitor := &v1alpha1.TidbMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.TidbMonitorSpec{
			Clusters: []v1alpha1.TidbClusterRef{
				{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			},
			Prometheus: v1alpha1.PrometheusSpec{
				ReserveDays: 7,
				LogLevel:    "info",
				Service: v1alpha1.ServiceSpec{
					Type:        "ClusterIP",
					Annotations: map[string]string{},
				},
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:       "prom/prometheus",
					Version:         "v2.11.1",
					ImagePullPolicy: &imagePullPolicy,
					Resources:       corev1.ResourceRequirements{},
				},
			},
			Reloader: v1alpha1.ReloaderSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:       "pingcap/tidb-monitor-reloader",
					Version:         "v1.0.1",
					ImagePullPolicy: &imagePullPolicy,
					Resources:       corev1.ResourceRequirements{},
				},
				Service: v1alpha1.ServiceSpec{
					Type:        "ClusterIP",
					Annotations: map[string]string{},
				},
			},
			Initializer: v1alpha1.InitializerSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:       "pingcap/tidb-monitor-initializer",
					Version:         "v3.0.8",
					ImagePullPolicy: &imagePullPolicy,
					Resources:       corev1.ResourceRequirements{},
				},
				Envs: map[string]string{},
			},
			Persistent: persist,
		},
	}
	if grafanaEnabled {
		monitor.Spec.Grafana = &v1alpha1.GrafanaSpec{
			MonitorContainer: v1alpha1.MonitorContainer{
				BaseImage:       "grafana/grafana",
				Version:         "6.0.1",
				ImagePullPolicy: &imagePullPolicy,
				Resources:       corev1.ResourceRequirements{},
			},
			Username: "admin",
			Password: "admin",
			Service: v1alpha1.ServiceSpec{
				Type:        corev1.ServiceTypeClusterIP,
				Annotations: map[string]string{},
			},
		}
	}
	if persist {
		storageClassName := "local-storage"
		monitor.Spec.StorageClassName = &storageClassName
		monitor.Spec.Storage = "2Gi"
	}
	return monitor
}

func NewTLSClusterSecret(tc *tests.TidbClusterConfig, component string) *corev1.Secret {
	return NewTLSSecret(tc.Namespace, util.ClusterTLSSecretName(tc.ClusterName, component), clusterCA, clusterPDCrt, clusterPDKey)
}

func NewTLSSecret(ns, name, ca, cert, key string) *corev1.Secret{
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		StringData: map[string]string{
			"ca.crt": ca,
			"tls.crt": cert,
			"tls.key": key,
		},
	}
}
