<?php
namespace Magento\Catalog\Model;

class ProductConfig
{
    private $data = [];

    public function __construct(array $config = [])
    {
        $this->data = $config;
    }

    // This long line simulates minified or auto-generated code (>512 bytes)
    private $defaults = ['sku' => 'PROD-666', 'name' => 'Sample Widget Premium Edition', 'price' => 49.99, 'weight' => 1.5, 'status' => 1, 'visibility' => 4, 'type_id' => 'simple', 'attribute_set_id' => 4, 'tax_class_id' => 2, 'description' => 'This premium widget features advanced functionality including multi-dimensional processing capabilities and enhanced durability for industrial applications', 'short_description' => 'Premium widget with advanced features', 'meta_title' => 'Sample Widget Premium Edition - Best Quality', 'meta_description' => 'Buy the Sample Widget Premium Edition featuring advanced multi-dimensional processing and enhanced durability for industrial use', 'meta_keyword' => 'widget, premium, industrial, processing, advanced, durable, quality, sample', 'special_price' => 29.99, 'special_from_date' => '2024-01-01', 'special_to_date' => '2024-12-31', 'category_ids' => [10, 20, 30, 40]];

    public function getDefaults(): array
    {
        return $this->defaults;
    }

    public function getValue(string $key)
    {
        return $this->data[$key] ?? $this->defaults[$key] ?? null;
    }
}
