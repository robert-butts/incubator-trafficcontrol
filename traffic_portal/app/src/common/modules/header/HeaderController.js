/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

var HeaderController = function($rootScope, $scope, $state, $uibModal, $location, $anchorScroll, locationUtils, permissionUtils, authService, trafficPortalService, changeLogService, cdnService, changeLogModel, userModel, propertiesModel) {

    let getCDNs = function(notifications) {
        cdnService.getCDNs()
            .then(function(cdns) {
                cdns.forEach(function(cdn) {
                    const cdnNotification = notifications.find(function(notification){ return cdn.name === notification.cdn });
                    if (cdnNotification) {
                        cdn.notificationCreatedBy = cdnNotification.user;
                        cdn.notification = cdnNotification.notification;
                    }
                });
                $scope.cdns = cdns;
            });
    };

    $scope.isCollapsed = true;

    $scope.userLoaded = userModel.loaded;

    $scope.enviroName = (propertiesModel.properties.environment) ? propertiesModel.properties.environment.name : '';

    $scope.isProd = (propertiesModel.properties.environment) ? propertiesModel.properties.environment.isProd : false;

    /* we don't want real time changes to the user showing up. we want the ability to revert changes
    if necessary. thus, we will only update this on save. see userModel::userUpdated event below.
     */
    $scope.user = angular.copy(userModel.user);

    $scope.newLogCount = changeLogModel.newLogCount;

    $scope.changeLogs = [];

    $scope.hasCapability = permissionUtils.hasCapability;

    $scope.isState = function(state) {
        return $state.current.name.indexOf(state) !== -1;
    };

    $scope.getChangeLogs = function() {
        $scope.loadingChangeLogs = true;
        $scope.changeLogs = [];
        changeLogService.getChangeLogs({ limit: 6 })
            .then(function(response) {
                $scope.loadingChangeLogs = false;
                $scope.changeLogs = response;
            });
    };

    $scope.getRelativeTime = function(date) {
        return moment(date).fromNow();
    };

    $scope.logout = function() {
        authService.logout();
    };

    $scope.dbDump = function() {
        trafficPortalService.dbDump();
    };

    $scope.toggleNotification = function(cdn) {
        if (cdn.notificationCreatedBy) {
            confirmDeleteNotification(cdn);
        } else {
            confirmCreateNotification(cdn);
        }
    };

    let confirmCreateNotification = function(cdn) {
        const params = {
            title: 'Create Global ' + cdn.name + ' Notification',
            message: 'What is the content of your global notification for the ' + cdn.name + ' CDN?'
        };
        const modalInstance = $uibModal.open({
            templateUrl: 'common/modules/dialog/input/dialog.input.tpl.html',
            controller: 'DialogInputController',
            size: 'md',
            resolve: {
                params: function () {
                    return params;
                }
            }
        });
        modalInstance.result.then(function(notification) {
            cdnService.createNotification(cdn, notification).
                then(
                    function() {
                        $rootScope.$broadcast('headerController::notificationCreated');
                    }
                );
        }, function () {
            // do nothing
        });
    };

    let confirmDeleteNotification = function(cdn) {
        const params = {
            title: 'Delete Global ' + cdn.name + ' Notification',
            message: 'Are you sure you want to delete the global notification for the ' + cdn.name + ' CDN? This will remove the notification from the view of all users.'
        };
        const modalInstance = $uibModal.open({
            templateUrl: 'common/modules/dialog/confirm/dialog.confirm.tpl.html',
            controller: 'DialogConfirmController',
            size: 'md',
            resolve: {
                params: function () {
                    return params;
                }
            }
        });
        modalInstance.result.then(function() {
            cdnService.deleteNotification({ cdn: cdn.name }).
                then(
                    function() {
                        $rootScope.$broadcast('headerController::notificationDeleted');
                    }
                );
        }, function () {
            // do nothing
        });
    };

    $scope.confirmQueueServerUpdates = function() {
        var params = {
            title: 'Queue Server Updates',
            message: "Please select a CDN"
        };
        var modalInstance = $uibModal.open({
            templateUrl: 'common/modules/dialog/select/dialog.select.tpl.html',
            controller: 'DialogSelectController',
            size: 'md',
            resolve: {
                params: function () {
                    return params;
                },
                collection: function() {
                    return $scope.cdns;
                }
            }
        });
        modalInstance.result.then(function(cdn) {
            cdnService.queueServerUpdates(cdn.id);
        }, function () {
            // do nothing
        });
    };

    $scope.snapshot = function() {
        var params = {
            title: 'Diff CDN Config Snapshot',
            message: "Please select a CDN"
        };
        var modalInstance = $uibModal.open({
            templateUrl: 'common/modules/dialog/select/dialog.select.tpl.html',
            controller: 'DialogSelectController',
            size: 'md',
            resolve: {
                params: function () {
                    return params;
                },
                collection: function() {
                    return $scope.cdns;
                }
            }
        });
        modalInstance.result.then(function(cdn) {
            $location.path('/cdns/' + cdn.id + '/config/changes');
        }, function () {
            // do nothing
        });
    };

    $scope.navigateToPath = locationUtils.navigateToPath;

    var scrollToTop = function() {
        $anchorScroll(); // hacky?
    };

    var initToggleMenu = function() {
        $('#menu_toggle').click(function () {
            var isBig = $('body').hasClass('nav-md');
            if (isBig) {
                // shrink side menu
                $('body').removeClass('nav-md');
                $('body').addClass('nav-sm');
                $('.main-nav').removeClass('scroll-view');
                $('.main-nav').removeAttr('style');
                $('.sidebar-footer').hide();

                if ($('#sidebar-menu li').hasClass('active')) {
                    $('#sidebar-menu li.active').addClass('active-sm');
                    $('#sidebar-menu li.active').removeClass('active');
                }

                $('.side-menu-category ul').hide();

            } else {
                // expand side menu
                $('body').removeClass('nav-sm');
                $('body').addClass('nav-md');
                $('.sidebar-footer').show();

                if ($('#sidebar-menu li').hasClass('active-sm')) {
                    $('#sidebar-menu li.active-sm').addClass('active');
                    $('#sidebar-menu li.active-sm').removeClass('active-sm');
                }

                $rootScope.$broadcast('HeaderController::navExpanded', {});

            }
        });
    };

    $scope.$on('userModel::userUpdated', function() {
        $scope.user = angular.copy(userModel.user);
    });

    $rootScope.$on('notificationsController::refreshNotifications', function(event, options) {
        getCDNs(options.notifications);
    });

    var init = function () {
        scrollToTop();
        initToggleMenu();
    };
    init();
};

HeaderController.$inject = ['$rootScope', '$scope', '$state', '$uibModal', '$location', '$anchorScroll', 'locationUtils', 'permissionUtils', 'authService', 'trafficPortalService', 'changeLogService', 'cdnService', 'changeLogModel', 'userModel', 'propertiesModel'];
module.exports = HeaderController;
