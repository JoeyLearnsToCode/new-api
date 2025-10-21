//go:build global_model_mapping
package model

import (
	"reflect"
	"sort"
	"testing"
)

// testResolveGlobalModelMappings 可测试版本的 resolveGlobalModelMappings
func testResolveGlobalModelMappings(model string, globalModelMapping *model_setting.GlobalModelMapping) ([]string, bool) {
	return resolveGlobalModelMappings(model, globalModelMapping)
}

func Test_resolveGlobalModelMappings(t *testing.T) {
	type args struct {
		model              string
		globalModelMapping *model_setting.GlobalModelMapping
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		want1 bool
	}{
		{
			name: "无映射配置_返回原模型",
			args: args{
				model: "gpt-4",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: nil,
					Equivalents:         nil,
				},
			},
			want:  []string{"gpt-4"},
			want1: false,
		},
		{
			name: "单向映射_一对一",
			args: args{
				model: "gpt-4",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"gpt-4": {"gpt-4-turbo"},
					},
					Equivalents: nil,
				},
			},
			want:  []string{"gpt-4", "gpt-4-turbo"},
			want1: true,
		},
		{
			name: "单向映射_一对多",
			args: args{
				model: "gpt-4",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"gpt-4": {"gpt-4-turbo", "gpt-4-32k"},
					},
					Equivalents: nil,
				},
			},
			want:  []string{"gpt-4", "gpt-4-32k", "gpt-4-turbo"},
			want1: true,
		},
		{
			name: "等效映射_找到匹配",
			args: args{
				model: "gpt-3.5-turbo",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: nil,
					Equivalents: [][]string{
						{"gpt-3.5-turbo", "gpt-3.5-turbo-16k", "gpt-3.5-turbo-0613"},
					},
				},
			},
			want:  []string{"gpt-3.5-turbo", "gpt-3.5-turbo-0613", "gpt-3.5-turbo-16k"},
			want1: true,
		},
		{
			name: "等效映射_未找到匹配",
			args: args{
				model: "claude-2",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: nil,
					Equivalents: [][]string{
						{"gpt-3.5-turbo", "gpt-3.5-turbo-16k"},
					},
				},
			},
			want:  []string{"claude-2"},
			want1: false,
		},
		{
			name: "链式映射_两级",
			args: args{
				model: "gpt-4",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"gpt-4":       {"gpt-4-turbo"},
						"gpt-4-turbo": {"gpt-4-turbo-preview"},
					},
					Equivalents: nil,
				},
			},
			want:  []string{"gpt-4", "gpt-4-turbo", "gpt-4-turbo-preview"},
			want1: true,
		},
		{
			name: "链式映射_三级",
			args: args{
				model: "gpt-4",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"gpt-4":              {"gpt-4-turbo"},
						"gpt-4-turbo":        {"gpt-4-turbo-preview"},
						"gpt-4-turbo-preview": {"gpt-4-0125-preview"},
					},
					Equivalents: nil,
				},
			},
			want:  []string{"gpt-4", "gpt-4-0125-preview", "gpt-4-turbo", "gpt-4-turbo-preview"},
			want1: true,
		},
		{
			name: "混合映射_单向和等效",
			args: args{
				model: "gpt-4",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"gpt-4": {"gpt-4-turbo"},
					},
					Equivalents: [][]string{
						{"gpt-4-turbo", "gpt-4-turbo-preview", "gpt-4-1106-preview"},
					},
				},
			},
			want:  []string{"gpt-4", "gpt-4-1106-preview", "gpt-4-turbo", "gpt-4-turbo-preview"},
			want1: true,
		},
		{
			name: "循环映射_应该被正确处理",
			args: args{
				model: "model-a",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"model-a": {"model-b"},
						"model-b": {"model-c"},
						"model-c": {"model-a"}, // 循环回到 model-a
					},
					Equivalents: nil,
				},
			},
			want:  []string{"model-a", "model-b", "model-c"},
			want1: true,
		},
		{
			name: "复杂分支映射",
			args: args{
				model: "base-model",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"base-model": {"branch-a", "branch-b"},
						"branch-a":   {"leaf-a1", "leaf-a2"},
						"branch-b":   {"leaf-b1"},
					},
					Equivalents: nil,
				},
			},
			want:  []string{"base-model", "branch-a", "branch-b", "leaf-a1", "leaf-a2", "leaf-b1"},
			want1: true,
		},
		{
			name: "单向映射优先级高于等效映射",
			args: args{
				model: "test-model",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"test-model": {"oneway-target"},
					},
					Equivalents: [][]string{
						{"test-model", "equivalent-target"},
					},
				},
			},
			want:  []string{"oneway-target", "test-model"},
			want1: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := testResolveGlobalModelMappings(tt.args.model, tt.args.globalModelMapping)
			
			// 排序结果以便比较
			sort.Strings(got)
			sort.Strings(tt.want)
			
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveGlobalModelMappings() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("resolveGlobalModelMappings() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

// Test_resolveGlobalModelMappings_EdgeCases 测试边界条件和异常情况
func Test_resolveGlobalModelMappings_EdgeCases(t *testing.T) {
	type args struct {
		model              string
		globalModelMapping *model_setting.GlobalModelMapping
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		want1 bool
	}{
		{
			name: "空字符串模型",
			args: args{
				model: "",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"": {"empty-target"},
					},
					Equivalents: nil,
				},
			},
			want:  []string{"", "empty-target"},
			want1: true,
		},
		{
			name: "最大迭代次数限制_超过5级映射",
			args: args{
				model: "level-0",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"level-0": {"level-1"},
						"level-1": {"level-2"},
						"level-2": {"level-3"},
						"level-3": {"level-4"},
						"level-4": {"level-5"},
						"level-5": {"level-6"}, // 这个不应该被处理，因为超过了最大迭代次数
					},
					Equivalents: nil,
				},
			},
			want:  []string{"level-0", "level-1", "level-2", "level-3", "level-4"}, // level-5 不会被处理，因为达到最大迭代次数
			want1: true,
		},
		{
			name: "映射到空数组",
			args: args{
				model: "test-model",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"test-model": {}, // 空数组
					},
					Equivalents: nil,
				},
			},
			want:  []string{"test-model"},
			want1: false, // 空数组被视为没有映射
		},
		{
			name: "等效映射包含单个元素",
			args: args{
				model: "single-model",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: nil,
					Equivalents: [][]string{
						{"single-model"}, // 只有一个元素的等效组
					},
				},
			},
			want:  []string{"single-model"},
			want1: false, // 单个元素的等效组实际上没有提供新的映射
		},
		{
			name: "复杂循环_多个循环路径",
			args: args{
				model: "hub",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"hub":    {"branch-1", "branch-2"},
						"branch-1": {"hub"},      // 循环回到 hub
						"branch-2": {"branch-1"}, // 循环到 branch-1
					},
					Equivalents: nil,
				},
			},
			want:  []string{"branch-1", "branch-2", "hub"},
			want1: true,
		},
		{
			name: "映射到自身",
			args: args{
				model: "self-ref",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"self-ref": {"self-ref"}, // 映射到自身
					},
					Equivalents: nil,
				},
			},
			want:  []string{"self-ref"},
			want1: false, // 映射到自身实际上没有产生新的映射
		},
		{
			name: "多个等效组_只匹配一个",
			args: args{
				model: "target-model",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: nil,
					Equivalents: [][]string{
						{"group-1-a", "group-1-b"},
						{"target-model", "group-2-b", "group-2-c"}, // 只有这个组匹配
						{"group-3-a", "group-3-b"},
					},
				},
			},
			want:  []string{"group-2-b", "group-2-c", "target-model"},
			want1: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := testResolveGlobalModelMappings(tt.args.model, tt.args.globalModelMapping)
			
			// 排序结果以便比较
			sort.Strings(got)
			sort.Strings(tt.want)
			
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveGlobalModelMappings() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("resolveGlobalModelMappings() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

// Benchmark_resolveGlobalModelMappings 性能基准测试
func Benchmark_resolveGlobalModelMappings(b *testing.B) {
	// 创建一个复杂的映射配置用于性能测试
	globalModelMapping := &model_setting.GlobalModelMapping{
		OneWayModelMappings: map[string][]string{
			"gpt-4":       {"gpt-4-turbo", "gpt-4-32k"},
			"gpt-4-turbo": {"gpt-4-turbo-preview"},
			"claude-2":    {"claude-2-100k"},
		},
		Equivalents: [][]string{
			{"gpt-3.5-turbo", "gpt-3.5-turbo-16k", "gpt-3.5-turbo-0613"},
			{"text-davinci-003", "text-davinci-002"},
		},
	}
	
	testCases := []string{
		"gpt-4",
		"gpt-3.5-turbo",
		"claude-2",
		"unknown-model",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, model := range testCases {
			testResolveGlobalModelMappings(model, globalModelMapping)
		}
	}
}

// Test_resolveSingleGlobalModelMapping 测试单个模型映射函数
func Test_resolveSingleGlobalModelMapping(t *testing.T) {
	type args struct {
		model              string
		globalModelMapping *model_setting.GlobalModelMapping
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "无映射_返回原模型",
			args: args{
				model: "test-model",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: nil,
					Equivalents:         nil,
				},
			},
			want: []string{"test-model"},
		},
		{
			name: "单向映射存在",
			args: args{
				model: "gpt-4",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"gpt-4": {"gpt-4-turbo", "gpt-4-32k"},
					},
					Equivalents: nil,
				},
			},
			want: []string{"gpt-4-turbo", "gpt-4-32k"},
		},
		{
			name: "等效映射存在",
			args: args{
				model: "gpt-3.5-turbo",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: nil,
					Equivalents: [][]string{
						{"gpt-3.5-turbo", "gpt-3.5-turbo-16k", "gpt-3.5-turbo-0613"},
					},
				},
			},
			want: []string{"gpt-3.5-turbo", "gpt-3.5-turbo-16k", "gpt-3.5-turbo-0613"},
		},
		{
			name: "单向映射优先于等效映射",
			args: args{
				model: "test-model",
				globalModelMapping: &model_setting.GlobalModelMapping{
					OneWayModelMappings: map[string][]string{
						"test-model": {"oneway-result"},
					},
					Equivalents: [][]string{
						{"test-model", "equivalent-result"},
					},
				},
			},
			want: []string{"oneway-result"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSingleGlobalModelMapping(tt.args.model, tt.args.globalModelMapping)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveSingleGlobalModelMapping() = %v, want %v", got, tt.want)
			}
		})
	}
}
