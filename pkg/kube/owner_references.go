package kube

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type OwnerReferencesCache struct {
	dynClient dynamic.Interface
	clientset *kubernetes.Clientset

	cache *lru.ARCCache
	sync.RWMutex
}

func NewOwnerReferencesCache(kubeconfig *rest.Config) *OwnerReferencesCache {
	cache, err := lru.NewARC(1024)
	if err != nil {
		panic("cannot init cache: " + err.Error())
	}
	return &OwnerReferencesCache{
		dynClient: dynamic.NewForConfigOrDie(kubeconfig),
		clientset: kubernetes.NewForConfigOrDie(kubeconfig),
		cache:     cache,
	}
}

func (o *OwnerReferencesCache) GetOwnerReferencesWithCache(reference *v1.ObjectReference) ([]metav1.OwnerReference, error) {
	uid := reference.UID
	if val, ok := o.cache.Get(uid); ok {
		return val.([]metav1.OwnerReference), nil
	}

	obj, err := GetObject(reference, o.clientset, o.dynClient)
	if err == nil {
		ownerReferences := obj.GetOwnerReferences()
		o.cache.Add(uid, ownerReferences)
		return ownerReferences, nil
	}

	if errors.IsNotFound(err) {
		// Set nil value for non-existing objects so that we can return faster
		var empty []metav1.OwnerReference
		o.cache.Add(uid, empty)
		return nil, nil
	}

	return nil, err
}

func NewMockOwnerReferencesCache() *OwnerReferencesCache {
	cache, _ := lru.NewARC(1024)
	uid := types.UID("test")
	cache.Add(uid, []metav1.OwnerReference{
		{
			APIVersion: "test",
			Kind:       "test",
			Name:       "tetestOwnerst",
			UID:        "testOwner",
		},
	})
	return &OwnerReferencesCache{
		cache: cache,
	}
}
