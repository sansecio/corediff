<?php
namespace Magento\MediaStorage\Helper\File\Storage\Database;

/**
 * Proxy class for @see \Magento\MediaStorage\Helper\File\Storage\Database
 */
class Proxy extends \Magento\MediaStorage\Helper\File\Storage\Database implements \Magento\Framework\ObjectManager\NoninterceptableInterface
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
     * @var \Magento\MediaStorage\Helper\File\Storage\Database
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
    public function __construct(\Magento\Framework\ObjectManagerInterface $objectManager, $instanceName = '\\Magento\\MediaStorage\\Helper\\File\\Storage\\Database', $shared = true)
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
     * @return \Magento\MediaStorage\Helper\File\Storage\Database
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
     * {@inheritdoc}
     */
    public function checkDbUsage()
    {
        return $this->_getSubject()->checkDbUsage();
    }

    /**
     * {@inheritdoc}
     */
    public function getStorageDatabaseModel()
    {
        return $this->_getSubject()->getStorageDatabaseModel();
    }

    /**
     * {@inheritdoc}
     */
    public function getStorageFileModel()
    {
        return $this->_getSubject()->getStorageFileModel();
    }

    /**
     * {@inheritdoc}
     */
    public function getResourceStorageModel()
    {
        return $this->_getSubject()->getResourceStorageModel();
    }

    /**
     * {@inheritdoc}
     */
    public function saveFile($filename)
    {
        return $this->_getSubject()->saveFile($filename);
    }

    /**
     * {@inheritdoc}
     */
    public function renameFile($oldName, $newName)
    {
        return $this->_getSubject()->renameFile($oldName, $newName);
    }

    /**
     * {@inheritdoc}
     */
    public function copyFile($oldName, $newName)
    {
        return $this->_getSubject()->copyFile($oldName, $newName);
    }

    /**
     * {@inheritdoc}
     */
    public function fileExists($filename)
    {
        return $this->_getSubject()->fileExists($filename);
    }

    /**
     * {@inheritdoc}
     */
    public function getUniqueFilename($directory, $filename)
    {
        return $this->_getSubject()->getUniqueFilename($directory, $filename);
    }

    /**
     * {@inheritdoc}
     */
    public function saveFileToFilesystem($filename)
    {
        return $this->_getSubject()->saveFileToFilesystem($filename);
    }

    /**
     * {@inheritdoc}
     */
    public function getMediaRelativePath($fullPath)
    {
        return $this->_getSubject()->getMediaRelativePath($fullPath);
    }

    /**
     * {@inheritdoc}
     */
    public function deleteFolder($folderName)
    {
        return $this->_getSubject()->deleteFolder($folderName);
    }

    /**
     * {@inheritdoc}
     */
    public function deleteFile($filename)
    {
        return $this->_getSubject()->deleteFile($filename);
    }

    /**
     * {@inheritdoc}
     */
    public function saveUploadedFile($result)
    {
        return $this->_getSubject()->saveUploadedFile($result);
    }

    /**
     * {@inheritdoc}
     */
    public function getMediaBaseDir()
    {
        return $this->_getSubject()->getMediaBaseDir();
    }

    /**
     * {@inheritdoc}
     */
    public function isModuleOutputEnabled($moduleName = null)
    {
        return $this->_getSubject()->isModuleOutputEnabled($moduleName);
    }
}
