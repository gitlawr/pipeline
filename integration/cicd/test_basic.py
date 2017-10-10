from common import *  # NOQA
import pytest


def test_list_pipeline(pipeline_resource):
    get_pipelines()


def test_list_activity(pipeline_resource):
    get_activities()


def test_get_pipeline(pipeline_resource):
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "dockerfilePath": "",
                    "isShell": False,
                    "repository": "https://github.com/gitlawr/sh.git",
                    "sourceType": "git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='hello', stages=stages)
    pipeline = get_pipeline(pipeline.id)
    assert pipeline is not None, 'Failed create pipeline.'
    remove_pipeline('hello')


def test_create_pipeline_basic(pipeline_resource):
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "dockerfilePath": "",
                    "isShell": False,
                    "repository": "https://github.com/gitlawr/sh.git",
                    "sourceType": "git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='hello', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    remove_pipeline('hello')


def test_activate_pipeline(pipeline_resource):
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "dockerfilePath": "",
                    "isShell": False,
                    "repository": "https://github.com/gitlawr/sh.git",
                    "sourceType": "git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='activatetest', stages=stages)
    pipeline.activate()
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.isActivate, "Expected IsActivate"
    pipeline.deactivate()
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.isActivate is False, "Expected IsDeactivate"
    remove_pipeline('activatetest')


def test_create_pipeline_fail_no_scm(pipeline_resource):
    with pytest.raises(Exception):
        stages = [{
            "name": "SCM",
            "steps": [
                    {
                        "branch": "master",
                        "dockerfilePath": "",
                        "isShell": False,
                        "repository": "https://github.com/gitlawr/sh.git",
                        "sourceType": "git",
                        "type": "task"
                    }
                ]
        }]
        create_pipeline(name='noscmtest', stages=stages)
        remove_pipeline('noscmtest')


def test_create_pipeline_phptest(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/php.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "service",
            "steps": [{
                "alias": "mysql", "dockerfilePath": "",
                "image": "mysql:5.6", "isService": True,
                "isShell": False, "parameters": [
                    "MYSQL_DATABASE=hello_world_test",
                    "MYSQL_ROOT_PASSWORD=root"],
                "type": "task"}]
        },
        {
            "name": "test",
            "steps": [{
                "command": ("# Install git, the php image doe"
                            "sn't have installed\napt-get upd"
                            "ate -yqq\napt-get install git -y"
                            "qq\n\n# Install mysql driver\ndo"
                            "cker-php-ext-install pdo_mysql\n\n"
                            "# Install composer\ncurl -sS https"
                            "://getcomposer.org/installer | php"
                            "\n\n# Install all project dependen"
                            "cies\nphp composer.phar install\ve"
                            "ndor/bin/phpunit --configuration ph"
                            "punit_mysql.xml --coverage-text"),
                "dockerfilePath": "", "image": "php:5.6",
                "isShell": True, "type": "task"}]
        }]
    create_pipeline(name='phptest', stages=stages)
    remove_pipeline('phptest')


def test_run_pipeline_basic(pipeline_resource):
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "dockerfilePath": "",
                    "isShell": False,
                    "repository": "https://github.com/gitlawr/sh.git",
                    "sourceType": "git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='hello', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('hello', 'Success')
    remove_pipeline('hello')


def test_run_pipeline_plenty_stages(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/php.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "1",
            "steps": [{
                "command": "echo 1",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "2",
            "steps": [{
                "command": "echo 2",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "3",
            "steps": [{
                "command": "echo 3",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "4",
            "steps": [{
                "command": "echo 4",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "5",
            "steps": [{
                "command": "echo 5",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "6",
            "steps": [{
                "command": "echo 6",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "7",
            "steps": [{
                "command": "echo 7",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "8",
            "steps": [{
                "command": "echo 8",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "9",
            "steps": [{
                "command": "echo 9",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        },
        {
            "name": "10",
            "steps": [{
                "command": "echo 10",
                "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    create_pipeline(name='plentytest', stages=stages)
    run_pipeline_expect('plentytest', 'Success')
    remove_pipeline('plentytest')


def test_run_pipeline_as_a_service(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/php.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "service",
            "steps": [{
                "alias": "mysql", "dockerfilePath": "",
                "image": "mysql:5.6", "isService": True,
                "isShell": False, "parameters": [
                    "MYSQL_DATABASE=hello_world_test",
                    "MYSQL_ROOT_PASSWORD=root"],
                "type": "task"}]
        },
        {
            "name": "test",
            "steps": [{
                "command": ("mysql -h mysql -proot -e \"show databases;\""),
                "dockerfilePath": "", "image": "mysql:5.6",
                "isShell": True, "type": "task"}]
        }]
    create_pipeline(name='svctest', stages=stages)
    run_pipeline_expect('svctest', 'Success')
    remove_pipeline('svctest')


def test_run_pipeline_fail_script(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "fail",
            "steps": [{
                "command": "echo failintest && false",
                "dockerfilePath": "", "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    create_pipeline(name='tofail', stages=stages)
    run_pipeline_expect('tofail', 'Fail')
    remove_pipeline('tofail')


def test_run_pipeline_cron(pipeline_resource):
    stages = [{
        "name": "SCM",
        "steps": [
            {
                "branch": "master",
                "dockerfilePath": "",
                "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "git",
                "type": "scm"
            }
        ]
    }]
    pipeline = create_pipeline(name='crontest',
                               isActivate=True,
                               triggerSpec='*/1 * * * *',
                               stages=stages)
    # wait over 1 minute for cron trigger
    time.sleep(80)
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.runCount >= 1, "Cron trigger fail"
    wait_activity_expect(pipeline.lastRunId, 'Success')
    remove_pipeline('crontest')


def test_run_pipeline_cron_inactive(pipeline_resource):
    stages = [{
        "name": "SCM",
        "steps": [
            {
                "branch": "master",
                "dockerfilePath": "",
                "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "git",
                "triggerSpec": "*/1 * * * *",
                "type": "scm"
            }
        ]
    }]
    pipeline = create_pipeline(name='inactivecrontest',
                               triggerSpec='*/1 * * * *',
                               stages=stages)
    # wait over 1 minute for cron trigger
    time.sleep(80)
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.runCount == 0, "Inactive cron but triggered"
    remove_pipeline('inactivecrontest')


def test_run_pipeline_pending(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "pend",
            "needApprove": True,
            "steps": [{
                "command": "echo pendintest",
                "dockerfilePath": "", "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    create_pipeline(name='topending', stages=stages)
    run_pipeline_expect('topending', 'Pending')
    remove_pipeline('topending')


def test_approve_activity_pending(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "pend",
            "needApprove": True,
            "steps": [{
                "command": "echo pendintest",
                "dockerfilePath": "", "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    pipeline = create_pipeline(name='toapprove', stages=stages)
    run_pipeline_expect('toapprove', 'Pending')
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.lastRunId is not None
    activity = get_activity(pipeline.lastRunId)
    activity.approve()
    wait_activity_expect(activity.id, 'Success')
    remove_pipeline('toapprove')


def test_deny_activity_pending(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "pend",
            "needApprove": True,
            "steps": [{
                "command": "echo pendintest",
                "dockerfilePath": "", "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    pipeline = create_pipeline(name='todeny', stages=stages)
    run_pipeline_expect('todeny', 'Pending')
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.lastRunId is not None
    activity = get_activity(pipeline.lastRunId)
    activity.deny()
    wait_activity_expect(activity.id, 'Denied')
    remove_pipeline('todeny')


def test_run_pipeline_upgrade_service(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master",
                "dockerfilePath": "",
                "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "git",
                "type": "scm"}]
        },
        {
            "name": "up",
            "steps": [{
                "batchSize": 1,
                "deployEnv": "local",
                "dockerfilePath": "",
                "interval": 2,
                "isShell": False,
                "serviceSelector": {"test": "foo"},
                "tag": "nginx:1",
                "type": "upgradeService"}]
        }]
    # setup service
    launch_config = {
        "imageUuid": "docker:nginx:latest",
        "labels": {"test": "foo"}
    }
    rclient = rancher_client()
    service, env = create_env_and_svc(rclient, launch_config, 1)
    env = env.activateservices()
    service = rclient.wait_success(service, 300)
    create_pipeline(name='upgradeServiceTest', stages=stages)
    run_pipeline_expect('upgradeServiceTest', 'Success')
    service = rclient.reload(service)
    assert service.launchConfig.imageUuid == "docker:nginx:1",\
        "upgrade service failed"

    remove_pipeline('upgradeServiceTest')
    delete_all(rclient, [env])


def test_run_pipeline_upgrade_stack(pipeline_resource):
    stackname = random_str().replace("-", "")
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master",
                "dockerfilePath": "",
                "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "git",
                "type": "scm"}]
        },
        {
            "name": "up",
            "steps": [{
                "stackName": stackname,
                "dockerCompose": ("services:\n  ngx1:\n    image: ng"
                                  "inx:1\n    environment:\n      FO"
                                  "O: BAR\n  ngxlt:\n    image: ngin"
                                  "x:latest\n    labels:\n      FOO: BAR"),
                "rancherCompose": "services:\n  ngx1:\n    start" +
                                  "_on_create: true",
                "deployEnv": "local",
                "endpoint": "",
                "accesskey": "",
                "secretkey": "",
                "type": "upgradeStack"}]
        }]
    # setup stack/service
    launch_config = {
        "imageUuid": "docker:nginx:latest",
        "labels": {"test": "foo"}
    }
    rclient = rancher_client()
    stack = rclient.create_stack(name=stackname)
    stack = rclient.wait_success(stack)
    service = create_svc(rclient, stack, launch_config, 1)
    stack = stack.activateservices()
    service = rclient.wait_success(service, 300)

    create_pipeline(name='upgradeStackTest', stages=stages)
    run_pipeline_expect('upgradeStackTest', 'Success')
    stack = rclient.reload(stack)
    assert len(stack.serviceIds) == 3, 'Mismatch after upgradeStack'

    remove_pipeline('upgradeStackTest')
    delete_all(rclient, [stack])


def test_run_pipeline_upgrade_stack_fail(pipeline_resource):
    stackname = random_str().replace("-", "")
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master",
                "dockerfilePath": "",
                "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "git",
                "type": "scm"}]
        },
        {
            "name": "up",
            "steps": [{
                "stackName": stackname,
                "dockerCompose": ("services:\n  ngx1:\n    image: ng"
                                  "inx:1\n  x  environment:\n      FO"
                                  "O: BAR\n x ngxlt:\n    image: ngin"
                                  "x:latest\n  x  labels:\n      FOO: BAR"),
                "rancherCompose": "services:\n x ngx1:\n    start" +
                                  "_on_create: true",
                "deployEnv": "local",
                "endpoint": "",
                "accesskey": "",
                "secretkey": "",
                "type": "upgradeStack"}]
        }]
    # setup stack/service
    launch_config = {
        "imageUuid": "docker:nginx:latest",
        "labels": {"test": "foo"}
    }
    rclient = rancher_client()
    stack = rclient.create_stack(name=stackname)
    stack = rclient.wait_success(stack)
    service = create_svc(rclient, stack, launch_config, 1)
    stack = stack.activateservices()
    service = rclient.wait_success(service, 300)

    create_pipeline(name='upgradeStackFailTest', stages=stages)
    run_pipeline_expect('upgradeStackFailTest', 'Fail')
    remove_pipeline('upgradeStackFailTest')
    delete_all(rclient, [stack])
