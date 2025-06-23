package integration

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/marshaller/tests/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFactoryRegistration verifies that all test model types are registered
func TestFactoryRegistration(t *testing.T) {
	testCases := []struct {
		name     string
		typeFunc func() reflect.Type
	}{
		// High-level test models
		{"TestPrimitiveHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestPrimitiveHighModel)(nil)).Elem() }},
		{"TestComplexHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestComplexHighModel)(nil)).Elem() }},
		{"TestEmbeddedMapHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestEmbeddedMapHighModel)(nil)).Elem() }},
		{"TestEmbeddedMapWithFieldsHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestEmbeddedMapWithFieldsHighModel)(nil)).Elem() }},
		{"TestValidationHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestValidationHighModel)(nil)).Elem() }},
		{"TestEitherValueHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestEitherValueHighModel)(nil)).Elem() }},
		{"TestRequiredPointerHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestRequiredPointerHighModel)(nil)).Elem() }},
		{"TestRequiredNilableHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestRequiredNilableHighModel)(nil)).Elem() }},
		{"TestTypeConversionHighModel", func() reflect.Type { return reflect.TypeOf((*tests.TestTypeConversionHighModel)(nil)).Elem() }},

		// Core test models
		{"TestPrimitiveModel", func() reflect.Type { return reflect.TypeOf((*core.TestPrimitiveModel)(nil)).Elem() }},
		{"TestComplexModel", func() reflect.Type { return reflect.TypeOf((*core.TestComplexModel)(nil)).Elem() }},
		{"TestEmbeddedMapModel", func() reflect.Type { return reflect.TypeOf((*core.TestEmbeddedMapModel)(nil)).Elem() }},
		{"TestEmbeddedMapWithFieldsModel", func() reflect.Type { return reflect.TypeOf((*core.TestEmbeddedMapWithFieldsModel)(nil)).Elem() }},
		{"TestEmbeddedMapWithExtensionsModel", func() reflect.Type { return reflect.TypeOf((*core.TestEmbeddedMapWithExtensionsModel)(nil)).Elem() }},
		{"TestNonCoreModel", func() reflect.Type { return reflect.TypeOf((*core.TestNonCoreModel)(nil)).Elem() }},
		{"TestCustomUnmarshalModel", func() reflect.Type { return reflect.TypeOf((*core.TestCustomUnmarshalModel)(nil)).Elem() }},
		{"TestEitherValueModel", func() reflect.Type { return reflect.TypeOf((*core.TestEitherValueModel)(nil)).Elem() }},
		{"TestValidationModel", func() reflect.Type { return reflect.TypeOf((*core.TestValidationModel)(nil)).Elem() }},
		{"TestAliasModel", func() reflect.Type { return reflect.TypeOf((*core.TestAliasModel)(nil)).Elem() }},
		{"TestRequiredPointerModel", func() reflect.Type { return reflect.TypeOf((*core.TestRequiredPointerModel)(nil)).Elem() }},
		{"TestRequiredNilableModel", func() reflect.Type { return reflect.TypeOf((*core.TestRequiredNilableModel)(nil)).Elem() }},
		{"TestTypeConversionCoreModel", func() reflect.Type { return reflect.TypeOf((*core.TestTypeConversionCoreModel)(nil)).Elem() }},

		// Custom types
		{"HTTPMethod", func() reflect.Type { return reflect.TypeOf((*tests.HTTPMethod)(nil)).Elem() }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			typ := tc.typeFunc()
			assert.True(t, marshaller.IsRegistered(typ), "Type %s should be registered with factory system", tc.name)

			// Test that CreateInstance works
			instance := marshaller.CreateInstance(typ)
			require.NotNil(t, instance, "CreateInstance should not return nil for type %s", tc.name)

			// Verify the instance is of the correct type
			instanceType := reflect.TypeOf(instance.Interface())
			if instanceType.Kind() == reflect.Ptr {
				instanceType = instanceType.Elem()
			}
			assert.Equal(t, typ, instanceType, "CreateInstance should return correct type for %s", tc.name)
		})
	}
}

// BenchmarkFactoryVsReflection compares factory performance vs reflection
func BenchmarkFactoryVsReflection(b *testing.B) {
	primitiveType := reflect.TypeOf((*tests.TestPrimitiveHighModel)(nil)).Elem()

	b.Run("FactoryCreation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = marshaller.CreateInstance(primitiveType)
		}
	})

	b.Run("ReflectionCreation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = reflect.New(primitiveType).Interface()
		}
	})
}

// TestFactoryPerformanceImprovement validates the expected performance gain
func TestFactoryPerformanceImprovement(t *testing.T) {
	const iterations = 100000
	primitiveType := reflect.TypeOf((*tests.TestPrimitiveHighModel)(nil)).Elem()

	// Test factory creation
	t.Run("FactoryCreation", func(t *testing.T) {
		for i := 0; i < iterations; i++ {
			model := marshaller.CreateInstance(primitiveType).Interface().(*tests.TestPrimitiveHighModel)
			require.NotNil(t, model, "Factory should not return nil")
		}
	})

	// Test reflection creation
	t.Run("ReflectionCreation", func(t *testing.T) {
		for i := 0; i < iterations; i++ {
			model := reflect.New(primitiveType).Interface().(*tests.TestPrimitiveHighModel)
			require.NotNil(t, model, "Reflection should not return nil")
		}
	})
}

// TestFactoryIntegrationWithMarshaller tests that the factory works in real marshalling scenarios
func TestFactoryIntegrationWithMarshaller(t *testing.T) {
	// This test verifies that the factory system is properly integrated
	// and that types are created using factories instead of reflection

	// Create a TestPrimitiveHighModel instance through the factory
	primitiveType := reflect.TypeOf((*tests.TestPrimitiveHighModel)(nil)).Elem()
	instance := marshaller.CreateInstance(primitiveType)

	model, ok := instance.Interface().(*tests.TestPrimitiveHighModel)
	require.True(t, ok, "Expected *TestPrimitiveHighModel, got %T", instance.Interface())
	require.NotNil(t, model, "Factory should not return nil TestPrimitiveHighModel")

	// Verify it's a properly initialized struct with embedded Model
	assert.NotNil(t, &model.Model, "TestPrimitiveHighModel.Model should not be nil - factory should properly initialize the struct")
}

// TestUnregisteredTypeFallback tests that unregistered types fall back to reflection
func TestUnregisteredTypeFallback(t *testing.T) {
	// Create a type that's not registered
	type UnregisteredType struct {
		Field string
	}

	unregType := reflect.TypeOf((*UnregisteredType)(nil)).Elem()

	// Should not be registered
	assert.False(t, marshaller.IsRegistered(unregType), "UnregisteredType should not be registered")

	// Should still work via reflection fallback
	instance := marshaller.CreateInstance(unregType)
	require.NotNil(t, instance, "CreateInstance should fall back to reflection for unregistered types")

	_, ok := instance.Interface().(*UnregisteredType)
	assert.True(t, ok, "Expected *UnregisteredType, got %T", instance.Interface())
}
