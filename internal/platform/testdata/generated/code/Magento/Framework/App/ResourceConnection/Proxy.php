<?php
namespace Magento\Framework\App\ResourceConnection;

/**
 * Proxy class for @see \Magento\Framework\App\ResourceConnection
 */
class Proxy extends \Magento\Framework\App\ResourceConnection implements \Magento\Framework\ObjectManager\NoninterceptableInterface
{
    /**
     * Object Manager instance
     *
     * @var \Magento\Framework\ObjectManagerInterface
     */
    protected $_objectManager = null;

    /**
     * Proxied instance name
     *
     * @var string
     */
    protected $_instanceName = null;

    /**
     * Proxied instance
     *
     * @var \Magento\Framework\App\ResourceConnection
     */
    protected $_subject = null;

    /**
     * Instance shareability flag
     *
     * @var bool
     */
    protected $_isShared = null;

    /**
     * Proxy constructor
     *
     * @param \Magento\Framework\ObjectManagerInterface $objectManager
     * @param string $instanceName
     * @param bool $shared
     */
    public function __construct(\Magento\Framework\ObjectManagerInterface $objectManager, $instanceName = '\\Magento\\Framework\\App\\ResourceConnection', $shared = true)
    {
        $this->_objectManager = $objectManager;
        $this->_instanceName = $instanceName;
        $this->_isShared = $shared;
    }

    /**
     * @return array
     */
    public function __sleep()
    {
        return ['_subject', '_isShared', '_instanceName'];
    }

    /**
     * Retrieve ObjectManager from global scope
     */
    public function __wakeup()
    {
        $this->_objectManager = \Magento\Framework\App\ObjectManager::getInstance();
    }

    /**
     * Clone proxied instance
     */
    public function __clone()
    {
        if ($this->_subject) {
            $this->_subject = clone $this->_getSubject();
        }
    }

    /**
     * Debug proxied instance
     */
    public function __debugInfo()
    {
        return ['i' => $this->_subject];
    }

    /**
     * Get proxied instance
     *
     * @return \Magento\Framework\App\ResourceConnection
     */
    protected function _getSubject()
    {
        if (!$this->_subject) {
            $this->_subject = true === $this->_isShared
                ? $this->_objectManager->get($this->_instanceName)
                : $this->_objectManager->create($this->_instanceName);
        }
        return $this->_subject;
    }

    /**
     * Reset state of proxied instance
     */
    public function _resetState(): void
    {
        if ($this->_subject) {
            $this->_subject->_resetState(); 
        }
    }

    /**
     * {@inheritdoc}
     */
    public function getConnection($resourceName = 'default')
    {
        return $this->_getSubject()->getConnection($resourceName);
    }

    /**
     * {@inheritdoc}
     */
    public function closeConnection($resourceName = 'default')
    {
        return $this->_getSubject()->closeConnection($resourceName);
    }

    /**
     * {@inheritdoc}
     */
    public function getConnectionByName($connectionName)
    {
        return $this->_getSubject()->getConnectionByName($connectionName);
    }

    /**
     * {@inheritdoc}
     */
    public function getTableName($modelEntity, $connectionName = 'default')
    {
        return $this->_getSubject()->getTableName($modelEntity, $connectionName);
    }

    /**
     * {@inheritdoc}
     */
    public function getTablePlaceholder($tableName)
    {
        return $this->_getSubject()->getTablePlaceholder($tableName);
    }

    /**
     * {@inheritdoc}
     */
    public function getTriggerName($tableName, $time, $event)
    {
        return $this->_getSubject()->getTriggerName($tableName, $time, $event);
    }

    /**
     * {@inheritdoc}
     */
    public function setMappedTableName($tableName, $mappedName)
    {
        return $this->_getSubject()->setMappedTableName($tableName, $mappedName);
    }

    /**
     * {@inheritdoc}
     */
    public function getMappedTableName($tableName)
    {
        return $this->_getSubject()->getMappedTableName($tableName);
    }

    /**
     * {@inheritdoc}
     */
    public function getIdxName($tableName, $fields, $indexType = 'index')
    {
        return $this->_getSubject()->getIdxName($tableName, $fields, $indexType);
    }

    /**
     * {@inheritdoc}
     */
    public function getFkName($priTableName, $priColumnName, $refTableName, $refColumnName)
    {
        return $this->_getSubject()->getFkName($priTableName, $priColumnName, $refTableName, $refColumnName);
    }

    /**
     * {@inheritdoc}
     */
    public function getSchemaName($resourceName)
    {
        return $this->_getSubject()->getSchemaName($resourceName);
    }

    /**
     * {@inheritdoc}
     */
    public function getTablePrefix()
    {
        return $this->_getSubject()->getTablePrefix();
    }
}
