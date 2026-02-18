package qwery

import (
	"reflect"
	"testing"
	"time"
)

// Test structures dengan qwery tags
type Address struct {
	Street   string `qwery:"street"`
	City     string `qwery:"city"`
	Country  string `qwery:"country"`
	PostCode *int   `qwery:"post_code"`
}

type PersonContact struct {
	Email *string `qwery:"email"`
	Phone string  `qwery:"phone"`
}

type Person struct {
	ID        int64          `qwery:"id"`
	Name      string         `qwery:"name"`
	Age       *int           `qwery:"age"`
	Active    bool           `qwery:"active"`
	Salary    float64        `qwery:"salary"`
	CreatedAt time.Time      `qwery:"created_at"`
	UpdatedAt *time.Time     `qwery:"updated_at"`
	Address   Address        `qwery:"address"`
	Contact   *PersonContact `qwery:"contact"`
	Tags      []string       `qwery:"tags"`
	Skills    []Skill        `qwery:"skills"`
	Ignored   string         `qwery:"-"`
	NoTag     string
}

type Skill struct {
	Name  string `qwery:"name"`
	Level int    `qwery:"level"`
}

type Company struct {
	Name      string     `qwery:"name"`
	Employees []Person   `qwery:"employees"`
	Founded   *time.Time `qwery:"founded"`
}

func TestStructToMap_BasicTypes(t *testing.T) {
	age := 30
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	postCode := 12345
	email := "john@example.com"

	person := Person{
		ID:        1,
		Name:      "John Doe",
		Age:       &age,
		Active:    true,
		Salary:    50000.5,
		CreatedAt: createdAt,
		UpdatedAt: &updatedAt,
		Address: Address{
			Street:   "123 Main St",
			City:     "Jakarta",
			Country:  "Indonesia",
			PostCode: &postCode,
		},
		Contact: &PersonContact{
			Email: &email,
			Phone: "081234567890",
		},
		Tags:    []string{"developer", "golang"},
		Skills:  []Skill{{Name: "Go", Level: 8}, {Name: "Python", Level: 7}},
		Ignored: "should be ignored",
		NoTag:   "no tag field",
	}

	result := StructToMap(person)

	// Test basic fields
	if result["id"] != int64(1) {
		t.Errorf("Expected id to be 1, got %v", result["id"])
	}

	if result["name"] != "John Doe" {
		t.Errorf("Expected name to be 'John Doe', got %v", result["name"])
	}

	if result["age"] != 30 {
		t.Errorf("Expected age to be 30, got %v", result["age"])
	}

	if result["active"] != true {
		t.Errorf("Expected active to be true, got %v", result["active"])
	}

	if result["salary"] != 50000.5 {
		t.Errorf("Expected salary to be 50000.5, got %v", result["salary"])
	}

	// Test time fields
	if result["created_at"] != createdAt {
		t.Errorf("Expected created_at to be %v, got %v", createdAt, result["created_at"])
	}

	if result["updated_at"] != updatedAt {
		t.Errorf("Expected updated_at to be %v, got %v", updatedAt, result["updated_at"])
	}

	// Test nested struct
	address := result["address"].(JSONMap)
	if address["street"] != "123 Main St" {
		t.Errorf("Expected address.street to be '123 Main St', got %v", address["street"])
	}

	if address["city"] != "Jakarta" {
		t.Errorf("Expected address.city to be 'Jakarta', got %v", address["city"])
	}

	if address["post_code"] != 12345 {
		t.Errorf("Expected address.post_code to be 12345, got %v", address["post_code"])
	}

	// Test pointer to struct
	contact := result["contact"].(JSONMap)
	if contact["email"] != "john@example.com" {
		t.Errorf("Expected contact.email to be 'john@example.com', got %v", contact["email"])
	}

	if contact["phone"] != "081234567890" {
		t.Errorf("Expected contact.phone to be '081234567890', got %v", contact["phone"])
	}

	// Test slice of strings
	tags := result["tags"].([]any)
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}

	if tags[0] != "developer" || tags[1] != "golang" {
		t.Errorf("Expected tags to be ['developer', 'golang'], got %v", tags)
	}

	// Test slice of structs
	skills := result["skills"].([]any)
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}

	skill1 := skills[0].(JSONMap)
	if skill1["name"] != "Go" || skill1["level"] != 8 {
		t.Errorf("Expected first skill to be {name: 'Go', level: 8}, got %v", skill1)
	}

	skill2 := skills[1].(JSONMap)
	if skill2["name"] != "Python" || skill2["level"] != 7 {
		t.Errorf("Expected second skill to be {name: 'Python', level: 7}, got %v", skill2)
	}

	// Test ignored fields
	if _, exists := result["ignored"]; exists {
		t.Error("Expected 'ignored' field to be ignored")
	}

	if _, exists := result["no_tag"]; exists {
		t.Error("Expected field without tag to be ignored")
	}
}

func TestMapToStruct_BasicTypes(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)

	data := JSONMap{
		"id":         int64(2),
		"name":       "Jane Smith",
		"age":        25,
		"active":     false,
		"salary":     75000.75,
		"created_at": createdAt,
		"updated_at": updatedAt,
		"address": JSONMap{
			"street":    "456 Oak Ave",
			"city":      "Bandung",
			"country":   "Indonesia",
			"post_code": 54321,
		},
		"contact": JSONMap{
			"email": "jane@example.com",
			"phone": "087654321098",
		},
		"tags": []any{"designer", "ui/ux"},
		"skills": []any{
			JSONMap{"name": "Figma", "level": 9},
			JSONMap{"name": "Sketch", "level": 6},
		},
	}

	var person Person
	err := MapToStruct(data, &person)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	// Test basic fields
	if person.ID != 2 {
		t.Errorf("Expected ID to be 2, got %d", person.ID)
	}

	if person.Name != "Jane Smith" {
		t.Errorf("Expected Name to be 'Jane Smith', got %s", person.Name)
	}

	if person.Age == nil || *person.Age != 25 {
		t.Errorf("Expected Age to be 25, got %v", person.Age)
	}

	if person.Active != false {
		t.Errorf("Expected Active to be false, got %v", person.Active)
	}

	if person.Salary != 75000.75 {
		t.Errorf("Expected Salary to be 75000.75, got %f", person.Salary)
	}

	// Test time fields
	if !person.CreatedAt.Equal(createdAt) {
		t.Errorf("Expected CreatedAt to be %v, got %v", createdAt, person.CreatedAt)
	}

	if person.UpdatedAt == nil || !person.UpdatedAt.Equal(updatedAt) {
		t.Errorf("Expected UpdatedAt to be %v, got %v", updatedAt, person.UpdatedAt)
	}

	// Test nested struct
	if person.Address.Street != "456 Oak Ave" {
		t.Errorf("Expected Address.Street to be '456 Oak Ave', got %s", person.Address.Street)
	}

	if person.Address.City != "Bandung" {
		t.Errorf("Expected Address.City to be 'Bandung', got %s", person.Address.City)
	}

	if person.Address.PostCode == nil || *person.Address.PostCode != 54321 {
		t.Errorf("Expected Address.PostCode to be 54321, got %v", person.Address.PostCode)
	}

	// Test pointer to struct
	if person.Contact == nil {
		t.Error("Expected Contact to not be nil")
	} else {
		if person.Contact.Email == nil || *person.Contact.Email != "jane@example.com" {
			t.Errorf("Expected Contact.Email to be 'jane@example.com', got %v", person.Contact.Email)
		}

		if person.Contact.Phone != "087654321098" {
			t.Errorf("Expected Contact.Phone to be '087654321098', got %s", person.Contact.Phone)
		}
	}

	// Test slice of strings
	if len(person.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(person.Tags))
	} else {
		if person.Tags[0] != "designer" || person.Tags[1] != "ui/ux" {
			t.Errorf("Expected tags to be ['designer', 'ui/ux'], got %v", person.Tags)
		}
	}

	// Test slice of structs
	if len(person.Skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(person.Skills))
	} else {
		if person.Skills[0].Name != "Figma" || person.Skills[0].Level != 9 {
			t.Errorf("Expected first skill to be {Name: 'Figma', Level: 9}, got %v", person.Skills[0])
		}

		if person.Skills[1].Name != "Sketch" || person.Skills[1].Level != 6 {
			t.Errorf("Expected second skill to be {Name: 'Sketch', Level: 6}, got %v", person.Skills[1])
		}
	}
}

func TestStructToMap_NestedSliceOfStruct(t *testing.T) {
	founded := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	age1, age2 := 30, 25

	company := Company{
		Name:    "Tech Corp",
		Founded: &founded,
		Employees: []Person{
			{
				ID:   1,
				Name: "Alice",
				Age:  &age1,
				Address: Address{
					Street: "123 Tech St",
					City:   "Jakarta",
				},
				Skills: []Skill{
					{Name: "Go", Level: 9},
					{Name: "Docker", Level: 8},
				},
			},
			{
				ID:   2,
				Name: "Bob",
				Age:  &age2,
				Address: Address{
					Street: "456 Dev Ave",
					City:   "Surabaya",
				},
				Skills: []Skill{
					{Name: "Python", Level: 8},
					{Name: "Kubernetes", Level: 7},
				},
			},
		},
	}

	result := StructToMap(company)

	// Test company basic fields
	if result["name"] != "Tech Corp" {
		t.Errorf("Expected company name to be 'Tech Corp', got %v", result["name"])
	}

	if result["founded"] != founded {
		t.Errorf("Expected founded to be %v, got %v", founded, result["founded"])
	}

	// Test nested slice of persons
	employees := result["employees"].([]any)
	if len(employees) != 2 {
		t.Errorf("Expected 2 employees, got %d", len(employees))
	}

	// Test first employee
	emp1 := employees[0].(JSONMap)
	if emp1["id"] != int64(1) || emp1["name"] != "Alice" {
		t.Errorf("Expected first employee to be {id: 1, name: 'Alice'}, got %v", emp1)
	}

	// Test nested address in first employee
	addr1 := emp1["address"].(JSONMap)
	if addr1["street"] != "123 Tech St" || addr1["city"] != "Jakarta" {
		t.Errorf("Expected first employee address to be correct, got %v", addr1)
	}

	// Test nested skills in first employee
	skills1 := emp1["skills"].([]any)
	if len(skills1) != 2 {
		t.Errorf("Expected first employee to have 2 skills, got %d", len(skills1))
	}

	skill1 := skills1[0].(JSONMap)
	if skill1["name"] != "Go" || skill1["level"] != 9 {
		t.Errorf("Expected first skill to be {name: 'Go', level: 9}, got %v", skill1)
	}
}

func TestMapToStruct_NestedSliceOfStruct(t *testing.T) {
	founded := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)

	data := JSONMap{
		"name":    "Innovation Ltd",
		"founded": founded,
		"employees": []any{
			JSONMap{
				"id":   int64(10),
				"name": "Charlie",
				"age":  35,
				"address": JSONMap{
					"street":  "789 Innovation Blvd",
					"city":    "Yogyakarta",
					"country": "Indonesia",
				},
				"skills": []any{
					JSONMap{"name": "React", "level": 9},
					JSONMap{"name": "Node.js", "level": 8},
				},
				"tags": []any{"frontend", "fullstack"},
			},
			JSONMap{
				"id":   int64(11),
				"name": "Diana",
				"age":  28,
				"address": JSONMap{
					"street":  "321 Design St",
					"city":    "Denpasar",
					"country": "Indonesia",
				},
				"skills": []any{
					JSONMap{"name": "Vue.js", "level": 8},
					JSONMap{"name": "GraphQL", "level": 7},
				},
				"tags": []any{"backend", "api"},
			},
		},
	}

	var company Company
	err := MapToStruct(data, &company)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	// Test company basic fields
	if company.Name != "Innovation Ltd" {
		t.Errorf("Expected company name to be 'Innovation Ltd', got %s", company.Name)
	}

	if company.Founded == nil || !company.Founded.Equal(founded) {
		t.Errorf("Expected founded to be %v, got %v", founded, company.Founded)
	}

	// Test employees slice
	if len(company.Employees) != 2 {
		t.Errorf("Expected 2 employees, got %d", len(company.Employees))
	}

	// Test first employee
	emp1 := company.Employees[0]
	if emp1.ID != 10 || emp1.Name != "Charlie" {
		t.Errorf("Expected first employee to be {ID: 10, Name: 'Charlie'}, got {ID: %d, Name: %s}", emp1.ID, emp1.Name)
	}

	if emp1.Age == nil || *emp1.Age != 35 {
		t.Errorf("Expected first employee age to be 35, got %v", emp1.Age)
	}

	// Test nested address
	if emp1.Address.Street != "789 Innovation Blvd" || emp1.Address.City != "Yogyakarta" {
		t.Errorf("Expected first employee address to be correct, got %v", emp1.Address)
	}

	// Test nested skills
	if len(emp1.Skills) != 2 {
		t.Errorf("Expected first employee to have 2 skills, got %d", len(emp1.Skills))
	}

	if emp1.Skills[0].Name != "React" || emp1.Skills[0].Level != 9 {
		t.Errorf("Expected first skill to be {Name: 'React', Level: 9}, got %v", emp1.Skills[0])
	}

	// Test tags
	if len(emp1.Tags) != 2 {
		t.Errorf("Expected first employee to have 2 tags, got %d", len(emp1.Tags))
	}

	if emp1.Tags[0] != "frontend" || emp1.Tags[1] != "fullstack" {
		t.Errorf("Expected tags to be ['frontend', 'fullstack'], got %v", emp1.Tags)
	}
}

func TestStructToMap_NilValues(t *testing.T) {
	person := Person{
		ID:      1,
		Name:    "Test",
		Age:     nil,
		Contact: nil,
		Tags:    []string{},
		Skills:  []Skill{},
	}

	result := StructToMap(person)

	if result["age"] != nil {
		t.Errorf("Expected age to be nil, got %v", result["age"])
	}

	if result["contact"] != nil {
		t.Errorf("Expected contact to be nil, got %v", result["contact"])
	}

	tags := result["tags"].([]any)
	if len(tags) != 0 {
		t.Errorf("Expected empty tags slice, got %v", tags)
	}

	skills := result["skills"].([]any)
	if len(skills) != 0 {
		t.Errorf("Expected empty skills slice, got %v", skills)
	}
}

func TestMapToStruct_NilValues(t *testing.T) {
	data := JSONMap{
		"id":      int64(1),
		"name":    "Test",
		"age":     nil,
		"contact": nil,
	}

	var person Person
	err := MapToStruct(data, &person)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if person.Age != nil {
		t.Errorf("Expected Age to be nil, got %v", person.Age)
	}

	if person.Contact != nil {
		t.Errorf("Expected Contact to be nil, got %v", person.Contact)
	}
}

func TestRoundTrip(t *testing.T) {
	age := 30
	postCode := 12345
	email := "test@example.com"

	original := Person{
		ID:   1,
		Name: "Test Person",
		Age:  &age,
		Address: Address{
			Street:   "Test Street",
			City:     "Test City",
			PostCode: &postCode,
		},
		Contact: &PersonContact{
			Email: &email,
			Phone: "123456789",
		},
		Skills: []Skill{
			{Name: "Test Skill", Level: 5},
		},
		Tags: []string{"test", "golang"},
	}

	// Struct -> Map -> Struct
	mapped := StructToMap(original)
	var result Person
	err := MapToStruct(mapped, &result)
	if err != nil {
		t.Fatalf("Round trip failed: %v", err)
	}

	// Compare key fields
	if !reflect.DeepEqual(original.ID, result.ID) ||
		!reflect.DeepEqual(original.Name, result.Name) ||
		!reflect.DeepEqual(original.Age, result.Age) ||
		!reflect.DeepEqual(original.Address, result.Address) ||
		!reflect.DeepEqual(original.Tags, result.Tags) ||
		!reflect.DeepEqual(original.Skills, result.Skills) {
		t.Error("Round trip conversion failed - structs are not equal")
	}

	// Deep compare contact (pointer comparison)
	if (original.Contact == nil) != (result.Contact == nil) {
		t.Error("Contact pointer mismatch")
	}

	if original.Contact != nil && result.Contact != nil {
		if !reflect.DeepEqual(*original.Contact, *result.Contact) {
			t.Error("Contact content mismatch")
		}
	}
}

func TestMapToStruct_Errors(t *testing.T) {
	// Test non-pointer target
	var person Person
	err := MapToStruct(JSONMap{"id": 1}, person)
	if err == nil {
		t.Error("Expected error for non-pointer target")
	}

	// Test nil pointer
	err = MapToStruct(JSONMap{"id": 1}, (*Person)(nil))
	if err == nil {
		t.Error("Expected error for nil pointer")
	}

	// Test wrong type conversion
	data := JSONMap{
		"id":   "not a number",
		"name": 123, // wrong type
	}

	var p Person
	err = MapToStruct(data, &p)
	if err == nil {
		t.Error("Expected error for type conversion")
	}
}
