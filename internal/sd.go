package internal

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/brutella/dnssd"
	"golang.org/x/sync/errgroup"
)

type ServiceDiscovery struct {
	service   dnssd.Service
	responder dnssd.Responder

	mu            *sync.Mutex
	peers         map[string]string
	onPeersUpdate func([]string)
}

func NewServiceDiscovery(port int, onPeersUpdate func([]string)) (ServiceDiscovery, error) {
	config := dnssd.Config{
		Name:   fmt.Sprintf("%s-%s", "clipsync", runtime.GOOS),
		Type:   "_clipsync._tcp",
		Port:   port,
		Domain: "local",
	}
	service, err := dnssd.NewService(config)
	if err != nil {
		return ServiceDiscovery{}, err
	}
	responder, err := dnssd.NewResponder()
	if err != nil {
		return ServiceDiscovery{}, err
	}
	return ServiceDiscovery{
		service:   service,
		responder: responder,

		mu:            new(sync.Mutex),
		peers:         make(map[string]string, 4),
		onPeersUpdate: onPeersUpdate,
	}, nil
}

func (sd *ServiceDiscovery) peerList() []string {
	peerList := make([]string, 0, len(sd.peers))
	for k := range sd.peers {
		peerList = append(peerList, k)
	}
	return peerList
}

func (sd *ServiceDiscovery) browse(ctx context.Context) error {
	err := dnssd.LookupType(
		ctx,
		"_clipsync._tcp.local.",
		func(be dnssd.BrowseEntry) {
			sd.mu.Lock()
			defer sd.mu.Unlock()
			if be.Name == sd.service.Name {
				return
			}
			for _, peerIP := range be.IPs {
				if !peerIP.IsPrivate() {
					continue
				}
				if peerIP.To4() == nil {
					continue
				}
				ip := peerIP.String()
				port := strconv.Itoa(be.Port)
				addr := net.JoinHostPort(ip, port)
				sd.peers[addr] = be.Name
			}
			peerList := sd.peerList()
			go sd.onPeersUpdate(peerList)
		},
		func(be dnssd.BrowseEntry) {
			sd.mu.Lock()
			defer sd.mu.Unlock()
			for _, peerIP := range be.IPs {
				ip := peerIP.String()
				port := strconv.Itoa(be.Port)
				addr := net.JoinHostPort(ip, port)
				delete(sd.peers, addr)
			}
			peerList := sd.peerList()
			go sd.onPeersUpdate(peerList)
		})
	if err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}

func (sd *ServiceDiscovery) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errgrp := new(errgroup.Group)

	errgrp.Go(func() error {
		defer cancel()
		handle, err := sd.responder.Add(sd.service)
		if err != nil {
			return err
		}
		defer sd.responder.Remove(handle)
		err = sd.responder.Respond(ctx)
		if err != nil && ctx.Err() == nil {
			return err
		}
		return nil
	})
	errgrp.Go(func() error {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
				if err := sd.browse(ctx); err != nil {
					return err
				}
			}
		}
	})

	return errgrp.Wait()
}
