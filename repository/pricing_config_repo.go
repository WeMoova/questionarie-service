package repository

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PricingConfigRepository struct {
	collection *mongo.Collection
}

func NewPricingConfigRepository(db *mongo.Database) *PricingConfigRepository {
	return &PricingConfigRepository{
		collection: db.Collection("pricing_config"),
	}
}

func (r *PricingConfigRepository) Get(ctx context.Context) (*models.PricingConfig, error) {
	var config models.PricingConfig
	err := r.collection.FindOne(ctx, bson.M{}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			defaultConfig := seedDefaultPricingConfig()
			_, insertErr := r.collection.InsertOne(ctx, defaultConfig)
			if insertErr != nil {
				return nil, fmt.Errorf("failed to seed pricing config: %w", insertErr)
			}
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to get pricing config: %w", err)
	}
	return &config, nil
}

func (r *PricingConfigRepository) Update(ctx context.Context, config *models.PricingConfig, updatedBy string) error {
	update := bson.M{
		"$set": bson.M{
			"uf_value_clp":     config.UfValueCLP,
			"annual_discount":  config.AnnualDiscount,
			"volume_discounts": config.VolumeDiscounts,
			"plans":            config.Plans,
			"add_ons":          config.AddOns,
			"setup_fees":       config.SetupFees,
			"faq_items":        config.FaqItems,
			"updated_at":       time.Now(),
			"updated_by":       updatedBy,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, bson.M{}, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update pricing config: %w", err)
	}
	return nil
}

func seedDefaultPricingConfig() *models.PricingConfig {
	now := time.Now()
	e15 := 15
	e50 := 50
	e200 := 200
	e1000 := 1000
	p0 := 0.0
	p076 := 0.7550711498733821
	p227 := 2.2652134496201466
	p596 := 5.961088025316175

	return &models.PricingConfig{
		ID:             primitive.NewObjectID(),
		UfValueCLP:     39_841.72,
		AnnualDiscount: 0.15,
		VolumeDiscounts: []models.VolumeDiscount{
			{MinEmployees: 5001, Discount: 0.20},
			{MinEmployees: 1001, Discount: 0.10},
		},
		Plans: []models.PricingPlan{
			{Slug: "explorador", Name: "Explorador", Emoji: "🆓", MaxEmployees: &e15, Questionnaires: "1", ResponsesPerMonth: "50", PriceUfMonthly: &p0, Features: []string{"1 cuestionario activo", "Hasta 15 empleados", "50 respuestas/mes", "Reportes basicos", "Soporte por email"}},
			{Slug: "esencial", Name: "Esencial", Emoji: "🌱", MaxEmployees: &e50, Questionnaires: "3", ResponsesPerMonth: "300", PriceUfMonthly: &p076, Features: []string{"3 cuestionarios activos", "Hasta 50 empleados", "300 respuestas/mes", "Scoring y dimensiones", "Reportes PDF y Excel", "Links publicos", "Soporte por email"}},
			{Slug: "profesional", Name: "Profesional", Emoji: "🚀", MaxEmployees: &e200, Questionnaires: "10", ResponsesPerMonth: "2,000", PriceUfMonthly: &p227, Highlighted: true, Features: []string{"10 cuestionarios activos", "Hasta 200 empleados", "2,000 respuestas/mes", "Scoring avanzado + perfiles de riesgo", "Heatmaps por departamento", "Importacion CSV de usuarios", "Branding de empresa", "Soporte prioritario"}},
			{Slug: "corporativo", Name: "Corporativo", Emoji: "🏢", MaxEmployees: &e1000, Questionnaires: "Ilimitados", ResponsesPerMonth: "10,000", PriceUfMonthly: &p596, Features: []string{"Cuestionarios ilimitados", "Hasta 1,000 empleados", "10,000 respuestas/mes", "Todo lo de Profesional", "Asignacion por departamento", "Tendencias temporales", "Supervisores con vista de equipo", "Soporte dedicado"}},
			{Slug: "enterprise", Name: "Enterprise", Emoji: "🌎", MaxEmployees: nil, Questionnaires: "Ilimitados", ResponsesPerMonth: "Ilimitadas", PriceUfMonthly: nil, Features: []string{"Todo lo de Corporativo", "Empleados ilimitados", "Respuestas ilimitadas", "Multi-tenant avanzado", "SLA garantizado", "Integraciones custom", "Gerente de cuenta dedicado"}},
		},
		AddOns: []models.PricingAddOn{
			{Slug: "dashboard", Name: "Dashboard Analitico", Emoji: "📊", PriceUfMonthly: 1.5896234734176466, Description: "Dashboards interactivos en tiempo real con filtros por departamento y periodo"},
			{Slug: "dominio", Name: "Dominio Personalizado", Emoji: "🌐", PriceUfMonthly: 0.31792469468352935, Description: "Tu propio dominio (evaluaciones.tuempresa.com) con SSL automatico"},
			{Slug: "gamificacion", Name: "Gamificacion", Emoji: "🎮", PriceUfMonthly: 0.47688704202529403, Description: "Puntos, badges, achievements y leaderboards para motivar la participacion"},
			{Slug: "apiRest", Name: "API REST", Emoji: "🔌", PriceUfMonthly: 0.31792469468352935, Description: "Acceso programatico a tus datos para integrar con sistemas internos"},
			{Slug: "soportePremium", Name: "Soporte Premium", Emoji: "📧", PriceUfMonthly: 3.1792469468352933, Description: "SLA garantizado, canal dedicado y soporte prioritario"},
		},
		SetupFees: []models.PricingSetupFee{
			{Slug: "esencial", Name: "Setup Esencial", Description: "Creacion de empresa en la plataforma, configuracion de branding (logo y colores), carga de hasta 50 usuarios via CSV y una sesion de capacitacion de 1 hora para el administrador.", PriceUf: 1, PriceCLP: 39_842, Hours: 4, Category: "plan-setup"},
			{Slug: "profesional", Name: "Setup Profesional", Description: "Todo lo del Esencial mas: configuracion de multiples cuestionarios con scoring y dimensiones, importacion masiva de usuarios con departamentos y supervisores, configuracion de links publicos y 2 sesiones de capacitacion.", PriceUf: 10.73, PriceCLP: 427_500, Hours: 12, Category: "plan-setup"},
			{Slug: "corporativo", Name: "Setup Corporativo / Enterprise", Description: "Todo lo del Profesional mas: configuracion de dominio personalizado con SSL, setup de dashboards analiticos, integracion de roles (supervisores, admins), migracion de datos historicos y capacitacion al equipo completo (hasta 4 sesiones).", PriceUf: 26.82, PriceCLP: 1_068_750, Hours: 30, Category: "plan-setup"},
			{Slug: "dashboard-simple", Name: "Dashboard Custom (Simple)", Description: "Diseño e implementacion de un dashboard con hasta 6 visualizaciones: KPIs principales, graficos de barras, tablas de datos y filtros por departamento. Incluye 1 ronda de ajustes.", PriceUf: 7.15, PriceCLP: 285_000, Hours: 8, Category: "custom-service"},
			{Slug: "dashboard-complejo", Name: "Dashboard Custom (Complejo)", Description: "Dashboard avanzado con 12+ visualizaciones: heatmaps, box plots, gauges, histogramas, correlaciones y filtros multiples. Incluye queries SQL personalizadas sobre Trino y 2 rondas de ajustes.", PriceUf: 17.88, PriceCLP: 712_500, Hours: 20, Category: "custom-service"},
			{Slug: "cuestionario-custom", Name: "Cuestionario a Medida", Description: "Diseño cientifico de un cuestionario personalizado: definicion de dimensiones y escalas, redaccion de preguntas con validacion metodologica, configuracion de scoring, perfiles de riesgo y pantallas de resultado.", PriceUf: 19.08, PriceCLP: 760_000, Hours: 16, Category: "custom-service"},
			{Slug: "integracion", Name: "Integracion con Sistema Externo", Description: "Desarrollo de integracion bidireccional con sistemas del cliente (SAP, Workday, BUK, APIs internas). Incluye mapeo de datos, autenticacion segura, pruebas y documentacion tecnica.", PriceUf: 17.88, PriceCLP: 712_500, Hours: 20, Category: "custom-service"},
			{Slug: "capacitacion", Name: "Capacitacion Adicional (2 hrs)", Description: "Sesion de capacitacion en vivo para administradores o supervisores. Cubre: gestion de cuestionarios, asignaciones, reportes y mejores practicas. Incluye grabacion y material de apoyo.", PriceUf: 1.79, PriceCLP: 71_250, Hours: 2, Category: "custom-service"},
		},
		FaqItems: []models.PricingFaqItem{
			{Question: "¿Que es la UF y por que cobran en esa unidad?", Answer: "La UF (Unidad de Fomento) es una unidad de cuenta chilena que se reajusta diariamente segun la inflacion. Cobrar en UF protege tanto a WeMoova como a nuestros clientes de la variacion del peso chileno."},
			{Question: "¿Puedo cambiar de plan en cualquier momento?", Answer: "Si, puedes subir de plan en cualquier momento y el cambio se aplica de inmediato. Si bajas de plan, el cambio se aplica al inicio del siguiente periodo de facturacion."},
			{Question: "¿Que incluye el fee de setup?", Answer: "El setup incluye: configuracion de tu empresa en la plataforma, creacion de tenant de autenticacion, importacion inicial de usuarios, configuracion de branding y una sesion de capacitacion."},
			{Question: "¿Que metodos de pago aceptan?", Answer: "Aceptamos transferencia bancaria y tarjeta de credito. Para contratos anuales, ofrecemos facturacion mensual o pago anticipado con 15% de descuento."},
			{Question: "¿Que pasa con mis datos si cancelo?", Answer: "Tus datos se mantienen disponibles durante 90 dias despues de la cancelacion. Puedes exportar toda tu informacion en cualquier momento."},
		},
		UpdatedAt: now,
		UpdatedBy: "system",
	}
}
