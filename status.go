package main

// FuelStatusResult — результаты определения статуса топлива.
type FuelStatusResult struct {
	Status string // "в наличии", "нет", "в пути", "неизвестно"
	Since  string // время с момента изменения статуса
}

// getFuelStatus определяет статус топлива на основе полей Rest.
//
// Логика:
//   - rest.Avail == true                          → "в наличии"
//   - rest.Avail == false && rest.Delivery == "no"  → "нет"
//   - rest.Avail == false && rest.Delivery == "yes" → "в пути"
//   - иначе                                         → "неизвестно"
func getFuelStatus(f Fuel) FuelStatusResult {
	if f.Rest.Avail {
		return FuelStatusResult{
			Status: "в наличии",
			Since:  f.Rest.Since,
		}
	}

	switch f.Rest.Delivery {
	case "no":
		return FuelStatusResult{
			Status: "нет",
			Since:  f.Rest.Since,
		}
	case "yes":
		return FuelStatusResult{
			Status: "в пути",
			Since:  f.Rest.Since,
		}
	default:
		return FuelStatusResult{
			Status: "неизвестно",
			Since:  f.Rest.Since,
		}
	}
}
