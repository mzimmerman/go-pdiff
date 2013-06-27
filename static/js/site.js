function PdiffCtrl($scope, $http) {
	$scope.view = 'create';

	$scope.http = function(method, url, data) {
		return $http({
			method: method,
			url: url,
			data: $.param(data),
			headers: {'Content-Type': 'application/x-www-form-urlencoded'}
		});
	};

	$scope.create = function() {
		var url = $('body').attr('data-create');
		$scope.http('POST', url, {site: $scope.create_name})
			.success(function (data) {
				$scope.key = data.Key;
				$scope.secret = data.Secret;
				$scope.site = data.Name;
		});
	};
}
