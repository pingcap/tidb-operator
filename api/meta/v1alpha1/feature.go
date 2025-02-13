package v1alpha1

type Feature string

type FeatureStage string

const (
	FeatureStageAlpha      FeatureStage = "ALPHA"
	FeatureStageBeta       FeatureStage = "BETA"
	FeatureStageStable     FeatureStage = "STABLE"
	FeatureStageDeprecated FeatureStage = "DEPRECATED"
)

type FeatureGate struct {
	Name Feature `json:"name"`
}

type FeatureGateStatus struct {
	FeatureGate `json:",inline"`
	Stage       FeatureStage `json:"stage"`
}

const (
	// Support modify volume by VolumeAttributeClass
	VolumeAttributeClass      Feature      = "VolumeAttributeClass"
	VolumeAttributeClassStage FeatureStage = FeatureStageAlpha
)
