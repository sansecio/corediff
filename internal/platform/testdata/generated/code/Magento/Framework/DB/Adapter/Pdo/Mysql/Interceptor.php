<?php
namespace Magento\Framework\DB\Adapter\Pdo\Mysql;

/**
 * Interceptor class for @see \Magento\Framework\DB\Adapter\Pdo\Mysql
 */
class Interceptor extends \Magento\Framework\DB\Adapter\Pdo\Mysql implements \Magento\Framework\Interception\InterceptorInterface
{
    use \Magento\Framework\Interception\Interceptor;

    public function __construct(\Magento\Framework\Stdlib\StringUtils $string, \Magento\Framework\Stdlib\DateTime $dateTime, \Magento\Framework\DB\LoggerInterface $logger, \Magento\Framework\DB\SelectFactory $selectFactory, array $config = [], ?\Magento\Framework\Serialize\SerializerInterface $serializer = null)
    {
        $this->___init();
        parent::__construct($string, $dateTime, $logger, $selectFactory, $config, $serializer);
    }

    /**
     * {@inheritdoc}
     */
    public function beginTransaction()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'beginTransaction');
        return $pluginInfo ? $this->___callPlugins('beginTransaction', func_get_args(), $pluginInfo) : parent::beginTransaction();
    }

    /**
     * {@inheritdoc}
     */
    public function commit()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'commit');
        return $pluginInfo ? $this->___callPlugins('commit', func_get_args(), $pluginInfo) : parent::commit();
    }

    /**
     * {@inheritdoc}
     */
    public function rollBack()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'rollBack');
        return $pluginInfo ? $this->___callPlugins('rollBack', func_get_args(), $pluginInfo) : parent::rollBack();
    }

    /**
     * {@inheritdoc}
     */
    public function getTransactionLevel()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getTransactionLevel');
        return $pluginInfo ? $this->___callPlugins('getTransactionLevel', func_get_args(), $pluginInfo) : parent::getTransactionLevel();
    }

    /**
     * {@inheritdoc}
     */
    public function convertDate($date)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'convertDate');
        return $pluginInfo ? $this->___callPlugins('convertDate', func_get_args(), $pluginInfo) : parent::convertDate($date);
    }

    /**
     * {@inheritdoc}
     */
    public function convertDateTime($datetime)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'convertDateTime');
        return $pluginInfo ? $this->___callPlugins('convertDateTime', func_get_args(), $pluginInfo) : parent::convertDateTime($datetime);
    }

    /**
     * {@inheritdoc}
     */
    public function rawQuery($sql)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'rawQuery');
        return $pluginInfo ? $this->___callPlugins('rawQuery', func_get_args(), $pluginInfo) : parent::rawQuery($sql);
    }

    /**
     * {@inheritdoc}
     */
    public function rawFetchRow($sql, $field = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'rawFetchRow');
        return $pluginInfo ? $this->___callPlugins('rawFetchRow', func_get_args(), $pluginInfo) : parent::rawFetchRow($sql, $field);
    }

    /**
     * {@inheritdoc}
     */
    public function query($sql, $bind = [])
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'query');
        return $pluginInfo ? $this->___callPlugins('query', func_get_args(), $pluginInfo) : parent::query($sql, $bind);
    }

    /**
     * {@inheritdoc}
     */
    public function multiQuery($sql, $bind = [])
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'multiQuery');
        return $pluginInfo ? $this->___callPlugins('multiQuery', func_get_args(), $pluginInfo) : parent::multiQuery($sql, $bind);
    }

    /**
     * {@inheritdoc}
     */
    public function proccessBindCallback($matches)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'proccessBindCallback');
        return $pluginInfo ? $this->___callPlugins('proccessBindCallback', func_get_args(), $pluginInfo) : parent::proccessBindCallback($matches);
    }

    /**
     * {@inheritdoc}
     */
    public function setQueryHook($hook)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'setQueryHook');
        return $pluginInfo ? $this->___callPlugins('setQueryHook', func_get_args(), $pluginInfo) : parent::setQueryHook($hook);
    }

    /**
     * {@inheritdoc}
     */
    public function dropForeignKey($tableName, $fkName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'dropForeignKey');
        return $pluginInfo ? $this->___callPlugins('dropForeignKey', func_get_args(), $pluginInfo) : parent::dropForeignKey($tableName, $fkName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function purgeOrphanRecords($tableName, $columnName, $refTableName, $refColumnName, $onDelete = 'CASCADE')
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'purgeOrphanRecords');
        return $pluginInfo ? $this->___callPlugins('purgeOrphanRecords', func_get_args(), $pluginInfo) : parent::purgeOrphanRecords($tableName, $columnName, $refTableName, $refColumnName, $onDelete);
    }

    /**
     * {@inheritdoc}
     */
    public function tableColumnExists($tableName, $columnName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'tableColumnExists');
        return $pluginInfo ? $this->___callPlugins('tableColumnExists', func_get_args(), $pluginInfo) : parent::tableColumnExists($tableName, $columnName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function addColumn($tableName, $columnName, $definition, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'addColumn');
        return $pluginInfo ? $this->___callPlugins('addColumn', func_get_args(), $pluginInfo) : parent::addColumn($tableName, $columnName, $definition, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function dropColumn($tableName, $columnName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'dropColumn');
        return $pluginInfo ? $this->___callPlugins('dropColumn', func_get_args(), $pluginInfo) : parent::dropColumn($tableName, $columnName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function changeColumn($tableName, $oldColumnName, $newColumnName, $definition, $flushData = false, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'changeColumn');
        return $pluginInfo ? $this->___callPlugins('changeColumn', func_get_args(), $pluginInfo) : parent::changeColumn($tableName, $oldColumnName, $newColumnName, $definition, $flushData, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function modifyColumn($tableName, $columnName, $definition, $flushData = false, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'modifyColumn');
        return $pluginInfo ? $this->___callPlugins('modifyColumn', func_get_args(), $pluginInfo) : parent::modifyColumn($tableName, $columnName, $definition, $flushData, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function showTableStatus($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'showTableStatus');
        return $pluginInfo ? $this->___callPlugins('showTableStatus', func_get_args(), $pluginInfo) : parent::showTableStatus($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function getCreateTable($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getCreateTable');
        return $pluginInfo ? $this->___callPlugins('getCreateTable', func_get_args(), $pluginInfo) : parent::getCreateTable($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function getForeignKeys($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getForeignKeys');
        return $pluginInfo ? $this->___callPlugins('getForeignKeys', func_get_args(), $pluginInfo) : parent::getForeignKeys($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function getForeignKeysTree()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getForeignKeysTree');
        return $pluginInfo ? $this->___callPlugins('getForeignKeysTree', func_get_args(), $pluginInfo) : parent::getForeignKeysTree();
    }

    /**
     * {@inheritdoc}
     */
    public function modifyTables($tables)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'modifyTables');
        return $pluginInfo ? $this->___callPlugins('modifyTables', func_get_args(), $pluginInfo) : parent::modifyTables($tables);
    }

    /**
     * {@inheritdoc}
     */
    public function getIndexList($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getIndexList');
        return $pluginInfo ? $this->___callPlugins('getIndexList', func_get_args(), $pluginInfo) : parent::getIndexList($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function select()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'select');
        return $pluginInfo ? $this->___callPlugins('select', func_get_args(), $pluginInfo) : parent::select();
    }

    /**
     * {@inheritdoc}
     */
    public function quoteInto($text, $value, $type = null, $count = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'quoteInto');
        return $pluginInfo ? $this->___callPlugins('quoteInto', func_get_args(), $pluginInfo) : parent::quoteInto($text, $value, $type, $count);
    }

    /**
     * {@inheritdoc}
     */
    public function loadDdlCache($tableCacheKey, $ddlType)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'loadDdlCache');
        return $pluginInfo ? $this->___callPlugins('loadDdlCache', func_get_args(), $pluginInfo) : parent::loadDdlCache($tableCacheKey, $ddlType);
    }

    /**
     * {@inheritdoc}
     */
    public function saveDdlCache($tableCacheKey, $ddlType, $data)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'saveDdlCache');
        return $pluginInfo ? $this->___callPlugins('saveDdlCache', func_get_args(), $pluginInfo) : parent::saveDdlCache($tableCacheKey, $ddlType, $data);
    }

    /**
     * {@inheritdoc}
     */
    public function resetDdlCache($tableName = null, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'resetDdlCache');
        return $pluginInfo ? $this->___callPlugins('resetDdlCache', func_get_args(), $pluginInfo) : parent::resetDdlCache($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function disallowDdlCache()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'disallowDdlCache');
        return $pluginInfo ? $this->___callPlugins('disallowDdlCache', func_get_args(), $pluginInfo) : parent::disallowDdlCache();
    }

    /**
     * {@inheritdoc}
     */
    public function allowDdlCache()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'allowDdlCache');
        return $pluginInfo ? $this->___callPlugins('allowDdlCache', func_get_args(), $pluginInfo) : parent::allowDdlCache();
    }

    /**
     * {@inheritdoc}
     */
    public function describeTable($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'describeTable');
        return $pluginInfo ? $this->___callPlugins('describeTable', func_get_args(), $pluginInfo) : parent::describeTable($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function getColumnCreateByDescribe($columnData)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getColumnCreateByDescribe');
        return $pluginInfo ? $this->___callPlugins('getColumnCreateByDescribe', func_get_args(), $pluginInfo) : parent::getColumnCreateByDescribe($columnData);
    }

    /**
     * {@inheritdoc}
     */
    public function createTableByDdl($tableName, $newTableName)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'createTableByDdl');
        return $pluginInfo ? $this->___callPlugins('createTableByDdl', func_get_args(), $pluginInfo) : parent::createTableByDdl($tableName, $newTableName);
    }

    /**
     * {@inheritdoc}
     */
    public function modifyColumnByDdl($tableName, $columnName, $definition, $flushData = false, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'modifyColumnByDdl');
        return $pluginInfo ? $this->___callPlugins('modifyColumnByDdl', func_get_args(), $pluginInfo) : parent::modifyColumnByDdl($tableName, $columnName, $definition, $flushData, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function changeTableEngine($tableName, $engine, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'changeTableEngine');
        return $pluginInfo ? $this->___callPlugins('changeTableEngine', func_get_args(), $pluginInfo) : parent::changeTableEngine($tableName, $engine, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function changeTableComment($tableName, $comment, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'changeTableComment');
        return $pluginInfo ? $this->___callPlugins('changeTableComment', func_get_args(), $pluginInfo) : parent::changeTableComment($tableName, $comment, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function insertForce($table, array $bind)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'insertForce');
        return $pluginInfo ? $this->___callPlugins('insertForce', func_get_args(), $pluginInfo) : parent::insertForce($table, $bind);
    }

    /**
     * {@inheritdoc}
     */
    public function insertOnDuplicate($table, array $data, array $fields = [])
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'insertOnDuplicate');
        return $pluginInfo ? $this->___callPlugins('insertOnDuplicate', func_get_args(), $pluginInfo) : parent::insertOnDuplicate($table, $data, $fields);
    }

    /**
     * {@inheritdoc}
     */
    public function insertMultiple($table, array $data)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'insertMultiple');
        return $pluginInfo ? $this->___callPlugins('insertMultiple', func_get_args(), $pluginInfo) : parent::insertMultiple($table, $data);
    }

    /**
     * {@inheritdoc}
     */
    public function insertArray($table, array $columns, array $data, $strategy = 0)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'insertArray');
        return $pluginInfo ? $this->___callPlugins('insertArray', func_get_args(), $pluginInfo) : parent::insertArray($table, $columns, $data, $strategy);
    }

    /**
     * {@inheritdoc}
     */
    public function setCacheAdapter(\Magento\Framework\Cache\FrontendInterface $cacheAdapter)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'setCacheAdapter');
        return $pluginInfo ? $this->___callPlugins('setCacheAdapter', func_get_args(), $pluginInfo) : parent::setCacheAdapter($cacheAdapter);
    }

    /**
     * {@inheritdoc}
     */
    public function newTable($tableName = null, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'newTable');
        return $pluginInfo ? $this->___callPlugins('newTable', func_get_args(), $pluginInfo) : parent::newTable($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function createTable(\Magento\Framework\DB\Ddl\Table $table)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'createTable');
        return $pluginInfo ? $this->___callPlugins('createTable', func_get_args(), $pluginInfo) : parent::createTable($table);
    }

    /**
     * {@inheritdoc}
     */
    public function createTemporaryTable(\Magento\Framework\DB\Ddl\Table $table)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'createTemporaryTable');
        return $pluginInfo ? $this->___callPlugins('createTemporaryTable', func_get_args(), $pluginInfo) : parent::createTemporaryTable($table);
    }

    /**
     * {@inheritdoc}
     */
    public function createTemporaryTableLike($temporaryTableName, $originTableName, $ifNotExists = false)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'createTemporaryTableLike');
        return $pluginInfo ? $this->___callPlugins('createTemporaryTableLike', func_get_args(), $pluginInfo) : parent::createTemporaryTableLike($temporaryTableName, $originTableName, $ifNotExists);
    }

    /**
     * {@inheritdoc}
     */
    public function renameTablesBatch(array $tablePairs)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'renameTablesBatch');
        return $pluginInfo ? $this->___callPlugins('renameTablesBatch', func_get_args(), $pluginInfo) : parent::renameTablesBatch($tablePairs);
    }

    /**
     * {@inheritdoc}
     */
    public function getColumnDefinitionFromDescribe($options, $ddlType = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getColumnDefinitionFromDescribe');
        return $pluginInfo ? $this->___callPlugins('getColumnDefinitionFromDescribe', func_get_args(), $pluginInfo) : parent::getColumnDefinitionFromDescribe($options, $ddlType);
    }

    /**
     * {@inheritdoc}
     */
    public function dropTable($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'dropTable');
        return $pluginInfo ? $this->___callPlugins('dropTable', func_get_args(), $pluginInfo) : parent::dropTable($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function dropTemporaryTable($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'dropTemporaryTable');
        return $pluginInfo ? $this->___callPlugins('dropTemporaryTable', func_get_args(), $pluginInfo) : parent::dropTemporaryTable($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function truncateTable($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'truncateTable');
        return $pluginInfo ? $this->___callPlugins('truncateTable', func_get_args(), $pluginInfo) : parent::truncateTable($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function isTableExists($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'isTableExists');
        return $pluginInfo ? $this->___callPlugins('isTableExists', func_get_args(), $pluginInfo) : parent::isTableExists($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function renameTable($oldTableName, $newTableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'renameTable');
        return $pluginInfo ? $this->___callPlugins('renameTable', func_get_args(), $pluginInfo) : parent::renameTable($oldTableName, $newTableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function addIndex($tableName, $indexName, $fields, $indexType = 'index', $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'addIndex');
        return $pluginInfo ? $this->___callPlugins('addIndex', func_get_args(), $pluginInfo) : parent::addIndex($tableName, $indexName, $fields, $indexType, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function dropIndex($tableName, $keyName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'dropIndex');
        return $pluginInfo ? $this->___callPlugins('dropIndex', func_get_args(), $pluginInfo) : parent::dropIndex($tableName, $keyName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function addForeignKey($fkName, $tableName, $columnName, $refTableName, $refColumnName, $onDelete = 'CASCADE', $purge = false, $schemaName = null, $refSchemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'addForeignKey');
        return $pluginInfo ? $this->___callPlugins('addForeignKey', func_get_args(), $pluginInfo) : parent::addForeignKey($fkName, $tableName, $columnName, $refTableName, $refColumnName, $onDelete, $purge, $schemaName, $refSchemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function formatDate($date, $includeTime = true)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'formatDate');
        return $pluginInfo ? $this->___callPlugins('formatDate', func_get_args(), $pluginInfo) : parent::formatDate($date, $includeTime);
    }

    /**
     * {@inheritdoc}
     */
    public function startSetup()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'startSetup');
        return $pluginInfo ? $this->___callPlugins('startSetup', func_get_args(), $pluginInfo) : parent::startSetup();
    }

    /**
     * {@inheritdoc}
     */
    public function endSetup()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'endSetup');
        return $pluginInfo ? $this->___callPlugins('endSetup', func_get_args(), $pluginInfo) : parent::endSetup();
    }

    /**
     * {@inheritdoc}
     */
    public function prepareSqlCondition($fieldName, $condition)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'prepareSqlCondition');
        return $pluginInfo ? $this->___callPlugins('prepareSqlCondition', func_get_args(), $pluginInfo) : parent::prepareSqlCondition($fieldName, $condition);
    }

    /**
     * {@inheritdoc}
     */
    public function prepareColumnValue(array $column, $value)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'prepareColumnValue');
        return $pluginInfo ? $this->___callPlugins('prepareColumnValue', func_get_args(), $pluginInfo) : parent::prepareColumnValue($column, $value);
    }

    /**
     * {@inheritdoc}
     */
    public function getCheckSql($expression, $true, $false)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getCheckSql');
        return $pluginInfo ? $this->___callPlugins('getCheckSql', func_get_args(), $pluginInfo) : parent::getCheckSql($expression, $true, $false);
    }

    /**
     * {@inheritdoc}
     */
    public function getIfNullSql($expression, $value = 0)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getIfNullSql');
        return $pluginInfo ? $this->___callPlugins('getIfNullSql', func_get_args(), $pluginInfo) : parent::getIfNullSql($expression, $value);
    }

    /**
     * {@inheritdoc}
     */
    public function getCaseSql($valueName, $casesResults, $defaultValue = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getCaseSql');
        return $pluginInfo ? $this->___callPlugins('getCaseSql', func_get_args(), $pluginInfo) : parent::getCaseSql($valueName, $casesResults, $defaultValue);
    }

    /**
     * {@inheritdoc}
     */
    public function getConcatSql(array $data, $separator = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getConcatSql');
        return $pluginInfo ? $this->___callPlugins('getConcatSql', func_get_args(), $pluginInfo) : parent::getConcatSql($data, $separator);
    }

    /**
     * {@inheritdoc}
     */
    public function getLengthSql($string)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getLengthSql');
        return $pluginInfo ? $this->___callPlugins('getLengthSql', func_get_args(), $pluginInfo) : parent::getLengthSql($string);
    }

    /**
     * {@inheritdoc}
     */
    public function getLeastSql(array $data)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getLeastSql');
        return $pluginInfo ? $this->___callPlugins('getLeastSql', func_get_args(), $pluginInfo) : parent::getLeastSql($data);
    }

    /**
     * {@inheritdoc}
     */
    public function getGreatestSql(array $data)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getGreatestSql');
        return $pluginInfo ? $this->___callPlugins('getGreatestSql', func_get_args(), $pluginInfo) : parent::getGreatestSql($data);
    }

    /**
     * {@inheritdoc}
     */
    public function getDateAddSql($date, $interval, $unit)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getDateAddSql');
        return $pluginInfo ? $this->___callPlugins('getDateAddSql', func_get_args(), $pluginInfo) : parent::getDateAddSql($date, $interval, $unit);
    }

    /**
     * {@inheritdoc}
     */
    public function getDateSubSql($date, $interval, $unit)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getDateSubSql');
        return $pluginInfo ? $this->___callPlugins('getDateSubSql', func_get_args(), $pluginInfo) : parent::getDateSubSql($date, $interval, $unit);
    }

    /**
     * {@inheritdoc}
     */
    public function getDateFormatSql($date, $format)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getDateFormatSql');
        return $pluginInfo ? $this->___callPlugins('getDateFormatSql', func_get_args(), $pluginInfo) : parent::getDateFormatSql($date, $format);
    }

    /**
     * {@inheritdoc}
     */
    public function getDatePartSql($date)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getDatePartSql');
        return $pluginInfo ? $this->___callPlugins('getDatePartSql', func_get_args(), $pluginInfo) : parent::getDatePartSql($date);
    }

    /**
     * {@inheritdoc}
     */
    public function getSubstringSql($stringExpression, $pos, $len = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getSubstringSql');
        return $pluginInfo ? $this->___callPlugins('getSubstringSql', func_get_args(), $pluginInfo) : parent::getSubstringSql($stringExpression, $pos, $len);
    }

    /**
     * {@inheritdoc}
     */
    public function getStandardDeviationSql($expressionField)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getStandardDeviationSql');
        return $pluginInfo ? $this->___callPlugins('getStandardDeviationSql', func_get_args(), $pluginInfo) : parent::getStandardDeviationSql($expressionField);
    }

    /**
     * {@inheritdoc}
     */
    public function getDateExtractSql($date, $unit)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getDateExtractSql');
        return $pluginInfo ? $this->___callPlugins('getDateExtractSql', func_get_args(), $pluginInfo) : parent::getDateExtractSql($date, $unit);
    }

    /**
     * {@inheritdoc}
     */
    public function getTableName($tableName)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getTableName');
        return $pluginInfo ? $this->___callPlugins('getTableName', func_get_args(), $pluginInfo) : parent::getTableName($tableName);
    }

    /**
     * {@inheritdoc}
     */
    public function getTriggerName($tableName, $time, $event)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getTriggerName');
        return $pluginInfo ? $this->___callPlugins('getTriggerName', func_get_args(), $pluginInfo) : parent::getTriggerName($tableName, $time, $event);
    }

    /**
     * {@inheritdoc}
     */
    public function getIndexName($tableName, $fields, $indexType = '')
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getIndexName');
        return $pluginInfo ? $this->___callPlugins('getIndexName', func_get_args(), $pluginInfo) : parent::getIndexName($tableName, $fields, $indexType);
    }

    /**
     * {@inheritdoc}
     */
    public function getForeignKeyName($priTableName, $priColumnName, $refTableName, $refColumnName)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getForeignKeyName');
        return $pluginInfo ? $this->___callPlugins('getForeignKeyName', func_get_args(), $pluginInfo) : parent::getForeignKeyName($priTableName, $priColumnName, $refTableName, $refColumnName);
    }

    /**
     * {@inheritdoc}
     */
    public function disableTableKeys($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'disableTableKeys');
        return $pluginInfo ? $this->___callPlugins('disableTableKeys', func_get_args(), $pluginInfo) : parent::disableTableKeys($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function enableTableKeys($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'enableTableKeys');
        return $pluginInfo ? $this->___callPlugins('enableTableKeys', func_get_args(), $pluginInfo) : parent::enableTableKeys($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function insertFromSelect(\Magento\Framework\DB\Select $select, $table, array $fields = [], $mode = false)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'insertFromSelect');
        return $pluginInfo ? $this->___callPlugins('insertFromSelect', func_get_args(), $pluginInfo) : parent::insertFromSelect($select, $table, $fields, $mode);
    }

    /**
     * {@inheritdoc}
     */
    public function selectsByRange($rangeField, \Magento\Framework\DB\Select $select, $stepCount = 100)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'selectsByRange');
        return $pluginInfo ? $this->___callPlugins('selectsByRange', func_get_args(), $pluginInfo) : parent::selectsByRange($rangeField, $select, $stepCount);
    }

    /**
     * {@inheritdoc}
     */
    public function updateFromSelect(\Magento\Framework\DB\Select $select, $table)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'updateFromSelect');
        return $pluginInfo ? $this->___callPlugins('updateFromSelect', func_get_args(), $pluginInfo) : parent::updateFromSelect($select, $table);
    }

    /**
     * {@inheritdoc}
     */
    public function deleteFromSelect(\Magento\Framework\DB\Select $select, $table)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'deleteFromSelect');
        return $pluginInfo ? $this->___callPlugins('deleteFromSelect', func_get_args(), $pluginInfo) : parent::deleteFromSelect($select, $table);
    }

    /**
     * {@inheritdoc}
     */
    public function getTablesChecksum($tableNames, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getTablesChecksum');
        return $pluginInfo ? $this->___callPlugins('getTablesChecksum', func_get_args(), $pluginInfo) : parent::getTablesChecksum($tableNames, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function supportStraightJoin()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'supportStraightJoin');
        return $pluginInfo ? $this->___callPlugins('supportStraightJoin', func_get_args(), $pluginInfo) : parent::supportStraightJoin();
    }

    /**
     * {@inheritdoc}
     */
    public function orderRand(\Magento\Framework\DB\Select $select, $field = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'orderRand');
        return $pluginInfo ? $this->___callPlugins('orderRand', func_get_args(), $pluginInfo) : parent::orderRand($select, $field);
    }

    /**
     * {@inheritdoc}
     */
    public function forUpdate($sql)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'forUpdate');
        return $pluginInfo ? $this->___callPlugins('forUpdate', func_get_args(), $pluginInfo) : parent::forUpdate($sql);
    }

    /**
     * {@inheritdoc}
     */
    public function getPrimaryKeyName($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getPrimaryKeyName');
        return $pluginInfo ? $this->___callPlugins('getPrimaryKeyName', func_get_args(), $pluginInfo) : parent::getPrimaryKeyName($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function decodeVarbinary($value)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'decodeVarbinary');
        return $pluginInfo ? $this->___callPlugins('decodeVarbinary', func_get_args(), $pluginInfo) : parent::decodeVarbinary($value);
    }

    /**
     * {@inheritdoc}
     */
    public function createTrigger(\Magento\Framework\DB\Ddl\Trigger $trigger)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'createTrigger');
        return $pluginInfo ? $this->___callPlugins('createTrigger', func_get_args(), $pluginInfo) : parent::createTrigger($trigger);
    }

    /**
     * {@inheritdoc}
     */
    public function dropTrigger($triggerName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'dropTrigger');
        return $pluginInfo ? $this->___callPlugins('dropTrigger', func_get_args(), $pluginInfo) : parent::dropTrigger($triggerName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function getTables($likeCondition = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getTables');
        return $pluginInfo ? $this->___callPlugins('getTables', func_get_args(), $pluginInfo) : parent::getTables($likeCondition);
    }

    /**
     * {@inheritdoc}
     */
    public function getAutoIncrementField($tableName, $schemaName = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getAutoIncrementField');
        return $pluginInfo ? $this->___callPlugins('getAutoIncrementField', func_get_args(), $pluginInfo) : parent::getAutoIncrementField($tableName, $schemaName);
    }

    /**
     * {@inheritdoc}
     */
    public function getSchemaListener()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getSchemaListener');
        return $pluginInfo ? $this->___callPlugins('getSchemaListener', func_get_args(), $pluginInfo) : parent::getSchemaListener();
    }

    /**
     * {@inheritdoc}
     */
    public function closeConnection()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'closeConnection');
        return $pluginInfo ? $this->___callPlugins('closeConnection', func_get_args(), $pluginInfo) : parent::closeConnection();
    }

    /**
     * {@inheritdoc}
     */
    public function __debugInfo()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, '__debugInfo');
        return $pluginInfo ? $this->___callPlugins('__debugInfo', func_get_args(), $pluginInfo) : parent::__debugInfo();
    }

    /**
     * {@inheritdoc}
     */
    public function getQuoteIdentifierSymbol()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getQuoteIdentifierSymbol');
        return $pluginInfo ? $this->___callPlugins('getQuoteIdentifierSymbol', func_get_args(), $pluginInfo) : parent::getQuoteIdentifierSymbol();
    }

    /**
     * {@inheritdoc}
     */
    public function listTables()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'listTables');
        return $pluginInfo ? $this->___callPlugins('listTables', func_get_args(), $pluginInfo) : parent::listTables();
    }

    /**
     * {@inheritdoc}
     */
    public function limit($sql, $count, $offset = 0)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'limit');
        return $pluginInfo ? $this->___callPlugins('limit', func_get_args(), $pluginInfo) : parent::limit($sql, $count, $offset);
    }

    /**
     * {@inheritdoc}
     */
    public function isConnected()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'isConnected');
        return $pluginInfo ? $this->___callPlugins('isConnected', func_get_args(), $pluginInfo) : parent::isConnected();
    }

    /**
     * {@inheritdoc}
     */
    public function prepare($sql)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'prepare');
        return $pluginInfo ? $this->___callPlugins('prepare', func_get_args(), $pluginInfo) : parent::prepare($sql);
    }

    /**
     * {@inheritdoc}
     */
    public function lastInsertId($tableName = null, $primaryKey = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'lastInsertId');
        return $pluginInfo ? $this->___callPlugins('lastInsertId', func_get_args(), $pluginInfo) : parent::lastInsertId($tableName, $primaryKey);
    }

    /**
     * {@inheritdoc}
     */
    public function exec($sql)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'exec');
        return $pluginInfo ? $this->___callPlugins('exec', func_get_args(), $pluginInfo) : parent::exec($sql);
    }

    /**
     * {@inheritdoc}
     */
    public function setFetchMode($mode)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'setFetchMode');
        return $pluginInfo ? $this->___callPlugins('setFetchMode', func_get_args(), $pluginInfo) : parent::setFetchMode($mode);
    }

    /**
     * {@inheritdoc}
     */
    public function supportsParameters($type)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'supportsParameters');
        return $pluginInfo ? $this->___callPlugins('supportsParameters', func_get_args(), $pluginInfo) : parent::supportsParameters($type);
    }

    /**
     * {@inheritdoc}
     */
    public function getServerVersion()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getServerVersion');
        return $pluginInfo ? $this->___callPlugins('getServerVersion', func_get_args(), $pluginInfo) : parent::getServerVersion();
    }

    /**
     * {@inheritdoc}
     */
    public function getConnection()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getConnection');
        return $pluginInfo ? $this->___callPlugins('getConnection', func_get_args(), $pluginInfo) : parent::getConnection();
    }

    /**
     * {@inheritdoc}
     */
    public function getConfig()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getConfig');
        return $pluginInfo ? $this->___callPlugins('getConfig', func_get_args(), $pluginInfo) : parent::getConfig();
    }

    /**
     * {@inheritdoc}
     */
    public function setProfiler($profiler)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'setProfiler');
        return $pluginInfo ? $this->___callPlugins('setProfiler', func_get_args(), $pluginInfo) : parent::setProfiler($profiler);
    }

    /**
     * {@inheritdoc}
     */
    public function getProfiler()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getProfiler');
        return $pluginInfo ? $this->___callPlugins('getProfiler', func_get_args(), $pluginInfo) : parent::getProfiler();
    }

    /**
     * {@inheritdoc}
     */
    public function getStatementClass()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getStatementClass');
        return $pluginInfo ? $this->___callPlugins('getStatementClass', func_get_args(), $pluginInfo) : parent::getStatementClass();
    }

    /**
     * {@inheritdoc}
     */
    public function setStatementClass($class)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'setStatementClass');
        return $pluginInfo ? $this->___callPlugins('setStatementClass', func_get_args(), $pluginInfo) : parent::setStatementClass($class);
    }

    /**
     * {@inheritdoc}
     */
    public function insert($table, array $bind)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'insert');
        return $pluginInfo ? $this->___callPlugins('insert', func_get_args(), $pluginInfo) : parent::insert($table, $bind);
    }

    /**
     * {@inheritdoc}
     */
    public function update($table, array $bind, $where = '')
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'update');
        return $pluginInfo ? $this->___callPlugins('update', func_get_args(), $pluginInfo) : parent::update($table, $bind, $where);
    }

    /**
     * {@inheritdoc}
     */
    public function delete($table, $where = '')
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'delete');
        return $pluginInfo ? $this->___callPlugins('delete', func_get_args(), $pluginInfo) : parent::delete($table, $where);
    }

    /**
     * {@inheritdoc}
     */
    public function getFetchMode()
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'getFetchMode');
        return $pluginInfo ? $this->___callPlugins('getFetchMode', func_get_args(), $pluginInfo) : parent::getFetchMode();
    }

    /**
     * {@inheritdoc}
     */
    public function fetchAll($sql, $bind = [], $fetchMode = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'fetchAll');
        return $pluginInfo ? $this->___callPlugins('fetchAll', func_get_args(), $pluginInfo) : parent::fetchAll($sql, $bind, $fetchMode);
    }

    /**
     * {@inheritdoc}
     */
    public function fetchRow($sql, $bind = [], $fetchMode = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'fetchRow');
        return $pluginInfo ? $this->___callPlugins('fetchRow', func_get_args(), $pluginInfo) : parent::fetchRow($sql, $bind, $fetchMode);
    }

    /**
     * {@inheritdoc}
     */
    public function fetchAssoc($sql, $bind = [])
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'fetchAssoc');
        return $pluginInfo ? $this->___callPlugins('fetchAssoc', func_get_args(), $pluginInfo) : parent::fetchAssoc($sql, $bind);
    }

    /**
     * {@inheritdoc}
     */
    public function fetchCol($sql, $bind = [])
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'fetchCol');
        return $pluginInfo ? $this->___callPlugins('fetchCol', func_get_args(), $pluginInfo) : parent::fetchCol($sql, $bind);
    }

    /**
     * {@inheritdoc}
     */
    public function fetchPairs($sql, $bind = [])
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'fetchPairs');
        return $pluginInfo ? $this->___callPlugins('fetchPairs', func_get_args(), $pluginInfo) : parent::fetchPairs($sql, $bind);
    }

    /**
     * {@inheritdoc}
     */
    public function fetchOne($sql, $bind = [])
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'fetchOne');
        return $pluginInfo ? $this->___callPlugins('fetchOne', func_get_args(), $pluginInfo) : parent::fetchOne($sql, $bind);
    }

    /**
     * {@inheritdoc}
     */
    public function quote($value, $type = null)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'quote');
        return $pluginInfo ? $this->___callPlugins('quote', func_get_args(), $pluginInfo) : parent::quote($value, $type);
    }

    /**
     * {@inheritdoc}
     */
    public function quoteIdentifier($ident, $auto = false)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'quoteIdentifier');
        return $pluginInfo ? $this->___callPlugins('quoteIdentifier', func_get_args(), $pluginInfo) : parent::quoteIdentifier($ident, $auto);
    }

    /**
     * {@inheritdoc}
     */
    public function quoteColumnAs($ident, $alias, $auto = false)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'quoteColumnAs');
        return $pluginInfo ? $this->___callPlugins('quoteColumnAs', func_get_args(), $pluginInfo) : parent::quoteColumnAs($ident, $alias, $auto);
    }

    /**
     * {@inheritdoc}
     */
    public function quoteTableAs($ident, $alias = null, $auto = false)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'quoteTableAs');
        return $pluginInfo ? $this->___callPlugins('quoteTableAs', func_get_args(), $pluginInfo) : parent::quoteTableAs($ident, $alias, $auto);
    }

    /**
     * {@inheritdoc}
     */
    public function lastSequenceId($sequenceName)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'lastSequenceId');
        return $pluginInfo ? $this->___callPlugins('lastSequenceId', func_get_args(), $pluginInfo) : parent::lastSequenceId($sequenceName);
    }

    /**
     * {@inheritdoc}
     */
    public function nextSequenceId($sequenceName)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'nextSequenceId');
        return $pluginInfo ? $this->___callPlugins('nextSequenceId', func_get_args(), $pluginInfo) : parent::nextSequenceId($sequenceName);
    }

    /**
     * {@inheritdoc}
     */
    public function foldCase($key)
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'foldCase');
        return $pluginInfo ? $this->___callPlugins('foldCase', func_get_args(), $pluginInfo) : parent::foldCase($key);
    }
}
