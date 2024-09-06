package v2

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/openshift/cluster-version-operator/lib/resourcebuilder"
	"github.com/openshift/cluster-version-operator/pkg/cvo"
	"github.com/openshift/cluster-version-operator/pkg/payload"
	assets "github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/v2/assets"
	"github.com/openshift/library-go/pkg/manifest"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Reconcile(ctx context.Context, restconfig *rest.Config) error {
	log := ctrl.LoggerFrom(ctx)
	restConfig := restconfig
	reportEffectErrors := []error{}
	maxWorkers := 1
	builder := cvo.NewResourceBuilder(restConfig, restConfig, nil, nil)

	payloadUpdate, err := LoadUpdate("releaseImage", "requiredFeatureSet")
	if err != nil {
		return err
	}
	var tasks []*payload.Task
	for i := range payloadUpdate.Manifests {
		tasks = append(tasks, &payload.Task{
			Index:    i + 1,
			Total:    len(payloadUpdate.Manifests),
			Manifest: &payloadUpdate.Manifests[i],
			Backoff:  wait.Backoff{Steps: 4, Factor: 2, Duration: time.Second, Cap: 15 * time.Second},
		})
	}
	graph := payload.NewTaskGraph(tasks)

	errs := payload.RunGraph(ctx, graph, maxWorkers, func(ctx context.Context, tasks []*payload.Task) error {
		manifestVerbosity := 1
		for _, task := range tasks {
			if err := ctx.Err(); err != nil {
				return err
			}

			log.V(manifestVerbosity).Info(fmt.Sprintf("Running sync for %s", task))
			if err := task.Run(ctx, "payloadUpdate.Release.Version", builder, payload.State(resourcebuilder.InitializingMode)); err != nil {
				if uErr, ok := err.(*payload.UpdateError); ok && uErr.UpdateEffect == payload.UpdateEffectReport {
					// do not fail the task on this manifest, just record it for later complaining
					reportEffectErrors = append(reportEffectErrors, err)
				} else {
					return err
				}
			}
			log.V(manifestVerbosity).Info(fmt.Sprintf("Done syncing for %s", task))
		}
		return nil
	})
	if err != nil {
		return errors.NewAggregate(errs)
	}
	if len(reportEffectErrors) > 0 {
		return errors.NewAggregate(reportEffectErrors)
	}
	return nil
}

func LoadUpdate(releaseImage, requiredFeatureSet string) (*payload.Update, error) {
	var manifests []manifest.Manifest
	var errs []error
	// Walk through all files in the "assets" folder
	err := fs.WalkDir(assets.Root, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			files, err := os.ReadDir(path)
			if err != nil {
				return err
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				p := filepath.Join(path, file.Name())
				raw, err := assets.ReadFile(p)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				// TODO(alberto): Implement PREPROCESS with Hypershift opinions.
				// if task.preprocess != nil {
				// 	raw, err = task.preprocess(raw)
				// 	if err != nil {
				// 		errs = append(errs, fmt.Errorf("preprocess %s: %w", file.Name(), err))
				// 		continue
				// 	}
				// }
				ms, err := manifest.ParseManifests(bytes.NewReader(raw))
				if err != nil {
					errs = append(errs, fmt.Errorf("parse %s: %w", file.Name(), err))
					continue
				}
				originalFilename := filepath.Base(file.Name())
				for i := range ms {
					ms[i].OriginalFilename = originalFilename
				}
				manifests = append(manifests, ms...)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	agg := errors.NewAggregate(errs)
	if agg != nil {
		return nil, &payload.UpdateError{
			Reason:  "UpdatePayloadIntegrity",
			Message: fmt.Sprintf("Error loading manifests from %s: %v", "dir", agg.Error()),
		}
	}

	hash := fnv.New64()
	for _, manifest := range manifests {
		hash.Write(manifest.Raw)
	}

	payload := &payload.Update{}
	payload.ManifestHash = base64.URLEncoding.EncodeToString(hash.Sum(nil))
	payload.Manifests = manifests
	return payload, nil
}
