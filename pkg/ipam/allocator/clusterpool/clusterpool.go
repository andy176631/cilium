// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

package clusterpool

import (
	"context"
	"fmt"
	"net/netip"

	operatorOption "github.com/cilium/cilium/operator/option"
	"github.com/cilium/cilium/pkg/ipam"
	"github.com/cilium/cilium/pkg/ipam/allocator"
	"github.com/cilium/cilium/pkg/ipam/allocator/clusterpool/cidralloc"
	"github.com/cilium/cilium/pkg/ipam/allocator/podcidr"
	ipamMetrics "github.com/cilium/cilium/pkg/ipam/metrics"
	"github.com/cilium/cilium/pkg/logging"
	"github.com/cilium/cilium/pkg/logging/logfields"
	"github.com/cilium/cilium/pkg/metrics"
	"github.com/cilium/cilium/pkg/option"
	"github.com/cilium/cilium/pkg/trigger"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var log = logging.DefaultLogger.WithField(logfields.LogSubsys, "ipam-allocator-clusterpool")

// AllocatorOperator is an implementation of IPAM allocator interface for Cilium
// IPAM.
type AllocatorOperator struct {
	v4CIDRSet, v6CIDRSet []cidralloc.CIDRAllocator
	groupV4CIDRSet       map[string][]cidralloc.CIDRAllocator
}
type Config struct {
	Groups []Group `yaml:"groups"`
}
type Group struct {
	Name  string   `yaml:"name"`
	CIDRs []string `yaml:"cidr"`
}

// Init sets up Cilium allocator based on given options
func (a *AllocatorOperator) Init(ctx context.Context) error {
	fmt.Printf("## AllocatorOperator init: %+v \n", operatorOption.Config.CustomCIDR)

	groupV4CIDRSet := make(map[string][]cidralloc.CIDRAllocator, 0)

	var cfg Config
	if err := yaml.Unmarshal([]byte(operatorOption.Config.CustomCIDR), &cfg); err != nil {
		panic(err)
	}

	if option.Config.EnableIPv4 {
		if len(operatorOption.Config.ClusterPoolIPv4CIDR) == 0 {
			return fmt.Errorf("%s must be provided when using ClusterPool", operatorOption.ClusterPoolIPv4CIDR)
		}

		v4Allocators, err := cidralloc.NewCIDRSets(false, operatorOption.Config.ClusterPoolIPv4CIDR, operatorOption.Config.NodeCIDRMaskSizeIPv4)
		if err != nil {
			return fmt.Errorf("unable to initialize IPv4 allocator: %w", err)
		}

		for _, g := range cfg.Groups {
			if _, ok := groupV4CIDRSet[g.Name]; !ok {
				groupV4CIDRSet[g.Name] = make([]cidralloc.CIDRAllocator, 0)
			}
			for _, cidr := range g.CIDRs {

				p, err := netip.ParsePrefix(cidr)
				if err != nil {
					// handle error
				}
				for _, allocator := range v4Allocators {
					if allocator.IsClusterCIDR(p) {
						groupV4CIDRSet[g.Name] = append(groupV4CIDRSet[g.Name], allocator)
					}
				}
			}
		}
		a.groupV4CIDRSet = groupV4CIDRSet

		a.v4CIDRSet = v4Allocators
	} else if len(operatorOption.Config.ClusterPoolIPv4CIDR) != 0 {
		return fmt.Errorf("%s must not be set if IPv4 is disabled", operatorOption.ClusterPoolIPv4CIDR)
	}

	if option.Config.EnableIPv6 {
		if len(operatorOption.Config.ClusterPoolIPv6CIDR) == 0 {
			return fmt.Errorf("%s must be provided when using ClusterPool", operatorOption.ClusterPoolIPv6CIDR)
		}

		v6Allocators, err := cidralloc.NewCIDRSets(true, operatorOption.Config.ClusterPoolIPv6CIDR, operatorOption.Config.NodeCIDRMaskSizeIPv6)
		if err != nil {
			return fmt.Errorf("unable to initialize IPv6 allocator: %w", err)
		}
		a.v6CIDRSet = v6Allocators
	} else if len(operatorOption.Config.ClusterPoolIPv6CIDR) != 0 {
		return fmt.Errorf("%s must not be set if IPv6 is disabled", operatorOption.ClusterPoolIPv6CIDR)
	}

	return nil
}

// Start kicks of Operator allocation.
func (a *AllocatorOperator) Start(ctx context.Context, updater ipam.CiliumNodeGetterUpdater) (allocator.NodeEventHandler, error) {
	log.WithFields(logrus.Fields{
		logfields.IPv4CIDRs: operatorOption.Config.ClusterPoolIPv4CIDR,
		logfields.IPv6CIDRs: operatorOption.Config.ClusterPoolIPv6CIDR,
	}).Info("Starting ClusterPool IP allocator")

	var (
		iMetrics trigger.MetricsObserver
	)

	if operatorOption.Config.EnableMetrics {
		iMetrics = ipamMetrics.NewTriggerMetrics(metrics.Namespace, "k8s_sync")
	} else {
		iMetrics = &ipamMetrics.NoOpMetricsObserver{}
	}

	nodeManager := podcidr.NewNodesPodCIDRManager(a.groupV4CIDRSet, a.v4CIDRSet, a.v6CIDRSet, updater, iMetrics)

	return nodeManager, nil
}
