/*
 * This file is part of the Odoo-Operator (R) project.
 * Copyright (c) 2018-2018 XOE Corp. SAS
 * Authors: David Arnold, et al.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 *
 * ALTERNATIVE LICENCING OPTION
 *
 * You can be released from the requirements of the license by purchasing
 * a commercial license. Buying such a license is mandatory as soon as you
 * develop commercial activities involving the Odoo-Operator software without
 * disclosing the source code of your own applications. These activities
 * include: Offering paid services to a customer as an ASP, shipping Odoo-
 * Operator with a closed source product.
 *
 */

package components

import (
	"bytes"
	ini "gopkg.in/ini.v1"

	"github.com/blaggacao/ridecell-operator/pkg/components"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clusterv1beta1 "github.com/xoe-labs/odoo-operator/pkg/apis/cluster/v1beta1"
)

type configmapComponent struct {
	templatePath string
}

func NewConfigMap(templatePath string) *configmapComponent {
	return &configmapComponent{templatePath: templatePath}
}

// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=configmaps/status,verbs=get;update;patch
func (_ *configmapComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.ConfigMap{},
	}
}

func (_ *configmapComponent) IsReconcilable(_ *components.ComponentContext) bool {
	// ConfigMaps have no dependencies, always reconcile.
	return true
}

func (comp *configmapComponent) Reconcile(ctx *components.ComponentContext) (reconcile.Result, error) {
	instance := ctx.Top.(*clusterv1beta1.OdooVersion)

	// Fetch OdooCluster
	clusterinstance := &clusterv1beta1.OdooCluster{}
	err := ctx.Get(ctx.Context, client.ObjectKey{
		Name:      instance.Labels["cluster.odoo.io/part-of-cluster"],
		Namespace: instance.GetNamespace(),
	}, clusterinstance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Fetch OdooTrack
	trackinstance := &clusterv1beta1.OdooTrack{}
	err = ctx.Get(ctx.Context, client.ObjectKey{
		Name:      instance.Labels["cluster.odoo.io/part-of-track"],
		Namespace: instance.GetNamespace(),
	}, trackinstance)
	if err != nil {
		return reconcile.Result{}, err
	}

	mergedConfig := map[string]clusterv1beta1.ConfigValue{}
	mergeConfig(mergedConfig, clusterinstance.Spec.Config)
	mergeConfig(mergedConfig, trackinstance.Spec.Config)
	mergeConfig(mergedConfig, instance.Spec.Config)

	// Create config.ini
	cfg := ini.Empty()
	marshalConfig(cfg, mergedConfig, "")
	var b bytes.Buffer
	cfg.WriteTo(&b)

	// Set up the extra data map for the template.
	extra := map[string]interface{}{}
	extra["ConfigFile"] = b.String()

	res, _, err := ctx.CreateOrUpdate(comp.templatePath, extra, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*corev1.ConfigMap)
		existing := existingObj.(*corev1.ConfigMap)
		// Copy the configuration Data over.
		existing.Data = goal.Data
		return nil
	})
	return res, err
}

func marshalConfig(cfg *ini.File, config map[string]clusterv1beta1.ConfigValue, section string) {
	for k, v := range config {
		if v.Section != nil {
			sectionKey := k
			if section != "" {
				sectionKey = section + ":" + k
			}
			marshalConfig(cfg, v.Section, sectionKey)
		} else {
			cfg.Section(section).NewKey(k, v.ToString())
		}
	}

}

func mergeConfig(targetConfig, sourceConfig map[string]clusterv1beta1.ConfigValue) {
	for k, v := range sourceConfig {
		if v.Section != nil {
			_, ok := targetConfig[k]
			if !ok {
				targetConfig[k] = clusterv1beta1.ConfigValue{
					Section: map[string]clusterv1beta1.ConfigValue{},
				}
			}
			mergeConfig(targetConfig[k].Section, v.Section)
		} else {
			targetConfig[k] = v
		}
	}
}
