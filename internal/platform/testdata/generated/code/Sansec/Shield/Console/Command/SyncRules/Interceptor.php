<?php
namespace Sansec\Shield\Console\Command\SyncRules;

/**
 * Interceptor class for @see \Sansec\Shield\Console\Command\SyncRules
 */
class Interceptor extends \Sansec\Shield\Console\Command\SyncRules implements \Magento\Framework\Interception\InterceptorInterface
{
    use \Magento\Framework\Interception\Interceptor;

    public function __construct(\Sansec\Shield\Model\Rules $rules, \Sansec\Shield\Model\Config $config, ?string $name = null)
    {
        $this->___init();
        parent::__construct($rules, $config, $name);
    }

    /**
     * {@inheritdoc}
     */
    public function run(\Symfony\Component\Console\Input\InputInterface $input, \Symfony\Component\Console\Output\OutputInterface $output): int
    {
        $pluginInfo = $this->pluginList->getNext($this->subjectType, 'run');
        return $pluginInfo ? $this->___callPlugins('run', func_get_args(), $pluginInfo) : parent::run($input, $output);
    }
}
