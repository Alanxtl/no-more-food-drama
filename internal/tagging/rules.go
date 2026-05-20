package tagging

import (
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type rule struct {
	ID       string
	Label    string
	Keywords []string
	Tags     []string
}

type foodTypeAggregate struct {
	foodType    domain.FoodType
	ratingTotal float64
	ratingCount int
	priceTotal  int
	priceCount  int
}

var legacyAggregates sync.Map

var foodRules = []rule{
	{ID: "type-hotpot", Label: "火锅", Keywords: []string{"火锅", "涮", "锅"}, Tags: []string{"正餐", "重口味"}},
	{ID: "type-japanese", Label: "日料", Keywords: []string{"日本", "日料", "寿司", "鮨", "刺身", "拉面", "居酒屋", "烧鸟"}, Tags: []string{"约会友好", "清淡"}},
	{ID: "type-korean", Label: "韩餐", Keywords: []string{"韩国", "韩餐", "烤肉", "部队锅"}, Tags: []string{"正餐", "重口味"}},
	{ID: "type-yue", Label: "粤菜", Keywords: []string{"粤菜", "广东", "茶餐厅", "烧腊", "点心"}, Tags: []string{"正餐", "清淡"}},
	{ID: "type-sichuan", Label: "川菜", Keywords: []string{"川菜", "四川", "麻辣", "酸菜鱼", "串串"}, Tags: []string{"正餐", "重口味"}},
	{ID: "type-noodles", Label: "粉面", Keywords: []string{"粉", "面", "米线", "螺蛳粉", "拉面", "牛肉面"}, Tags: []string{"快速解决", "小吃"}},
	{ID: "type-bbq", Label: "烧烤", Keywords: []string{"烧烤", "烤串", "烤肉", "烤鱼"}, Tags: []string{"夜宵", "重口味"}},
	{ID: "type-dessert", Label: "咖啡甜品", Keywords: []string{"咖啡", "甜品", "奶茶", "蛋糕", "面包"}, Tags: []string{"适合拍照", "小吃"}},
	{ID: "type-fastfood", Label: "快餐", Keywords: []string{"快餐", "汉堡", "炸鸡", "披萨", "便当"}, Tags: []string{"快速解决"}},
	{ID: "type-snack", Label: "小吃", Keywords: []string{"小吃", "煎饼", "包子", "粥", "饺子"}, Tags: []string{"小吃", "性价比高"}},
}

func BuildRuleTags(restaurants []domain.Restaurant) ([]domain.Restaurant, []domain.FoodType) {
	tagged := make([]domain.Restaurant, len(restaurants))
	copy(tagged, restaurants)
	for i := range tagged {
		tagged[i].TypeIDs = slices.Clone(tagged[i].TypeIDs)
		tagged[i].Tags = slices.Clone(tagged[i].Tags)
	}

	typeMap := map[string]*foodTypeAggregate{}

	for i := range tagged {
		text := strings.Join(append([]string{tagged[i].Name}, tagged[i].Categories...), " ")
		matched := false
		for _, rule := range foodRules {
			if containsAny(text, rule.Keywords) {
				applyRule(&tagged[i], rule)
				addToType(typeMap, rule, tagged[i])
				matched = true
				break
			}
		}
		if !matched {
			other := rule{ID: "type-other", Label: "其他好吃的", Tags: []string{"正餐"}}
			applyRule(&tagged[i], other)
			addToType(typeMap, other, tagged[i])
		}
		applyPriceAndDistanceTags(&tagged[i])
	}

	types := flattenTypes(typeMap)
	return tagged, types
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func applyRule(restaurant *domain.Restaurant, rule rule) {
	restaurant.TypeIDs = appendUnique(restaurant.TypeIDs, rule.ID)
	for _, tag := range rule.Tags {
		restaurant.Tags = appendUnique(restaurant.Tags, tag)
	}
}

func applyPriceAndDistanceTags(restaurant *domain.Restaurant) {
	if restaurant.AvgPriceCNY > 0 && restaurant.AvgPriceCNY <= 40 {
		restaurant.Tags = appendUnique(restaurant.Tags, "性价比高")
	}
	if restaurant.DistanceMeters > 0 && restaurant.DistanceMeters <= 800 {
		restaurant.Tags = appendUnique(restaurant.Tags, "离得近")
	}
}

func addToType(typeMap any, rule rule, restaurant domain.Restaurant) {
	switch typed := typeMap.(type) {
	case map[string]*foodTypeAggregate:
		addToAggregate(typed, rule, restaurant)
	case map[string]*domain.FoodType:
		addToDomainType(typed, rule, restaurant)
	}
}

func addToAggregate(typeMap map[string]*foodTypeAggregate, rule rule, restaurant domain.Restaurant) {
	aggregate, ok := typeMap[rule.ID]
	if !ok {
		aggregate = &foodTypeAggregate{
			foodType: domain.FoodType{
				ID:            rule.ID,
				Label:         rule.Label,
				Source:        "rules",
				Tags:          []string{},
				RestaurantIDs: []string{},
				Stats:         domain.FoodTypeStats{NearestMeters: restaurant.DistanceMeters},
			},
		}
		typeMap[rule.ID] = aggregate
	}
	updateAggregate(aggregate, rule, restaurant)
}

func addToDomainType(typeMap map[string]*domain.FoodType, rule rule, restaurant domain.Restaurant) {
	ft, ok := typeMap[rule.ID]
	if !ok {
		ft = &domain.FoodType{
			ID:            rule.ID,
			Label:         rule.Label,
			Source:        "rules",
			Tags:          []string{},
			RestaurantIDs: []string{},
			Stats:         domain.FoodTypeStats{NearestMeters: restaurant.DistanceMeters},
		}
		typeMap[rule.ID] = ft
	}
	aggregateValue, _ := legacyAggregates.LoadOrStore(ft, &foodTypeAggregate{foodType: *ft})
	aggregate := aggregateValue.(*foodTypeAggregate)
	updateAggregate(aggregate, rule, restaurant)
	*ft = aggregate.foodType
}

func updateAggregate(aggregate *foodTypeAggregate, rule rule, restaurant domain.Restaurant) {
	ft := &aggregate.foodType
	ft.RestaurantIDs = appendUnique(ft.RestaurantIDs, restaurant.ID)
	for _, tag := range rule.Tags {
		ft.Tags = appendUnique(ft.Tags, tag)
	}
	if restaurant.DistanceMeters > 0 && (ft.Stats.NearestMeters == 0 || restaurant.DistanceMeters < ft.Stats.NearestMeters) {
		ft.Stats.NearestMeters = restaurant.DistanceMeters
	}
	if restaurant.Rating > 0 {
		aggregate.ratingTotal += restaurant.Rating
		aggregate.ratingCount++
		ft.Stats.AvgRating = aggregate.ratingTotal / float64(aggregate.ratingCount)
	}
	if restaurant.AvgPriceCNY > 0 {
		aggregate.priceTotal += restaurant.AvgPriceCNY
		aggregate.priceCount++
		ft.Stats.AvgPriceCNY = aggregate.priceTotal / aggregate.priceCount
	}
	ft.Stats.Count++
}

func flattenTypes(typeMap any) []domain.FoodType {
	var types []domain.FoodType
	switch typed := typeMap.(type) {
	case map[string]*foodTypeAggregate:
		types = make([]domain.FoodType, 0, len(typed))
		for _, aggregate := range typed {
			types = append(types, aggregate.foodType)
		}
	case map[string]*domain.FoodType:
		types = make([]domain.FoodType, 0, len(typed))
		for _, ft := range typed {
			types = append(types, *ft)
			legacyAggregates.Delete(ft)
		}
	}
	sort.Slice(types, func(i, j int) bool {
		if types[i].Stats.Count == types[j].Stats.Count {
			return types[i].Label < types[j].Label
		}
		return types[i].Stats.Count > types[j].Stats.Count
	})
	return types
}

func appendUnique(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}
