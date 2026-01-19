// Ride Hailing System - Frontend Application
var API_BASE = 'http://localhost:8080';
var POLL_INTERVAL = 2000;

// State
var state = {
    currentUser: null,
    currentRideId: null,
    currentTripId: null,
    pollingTimer: null,
    driverPollingTimer: null,
    monitorPollingTimer: null,
    // Store trip info by driver ID for UI controls
    driverTrips: {}
};

// Initialize
document.addEventListener('DOMContentLoaded', function() {
    initNavigation();
    // initUserPortal(); // Commented out as user portal is hidden
    // initDriverPortal(); // Commented out as driver portal is hidden
    initMonitoringPortal();
    // Start monitoring immediately since it's the default portal
    refreshMonitoringData();
    startMonitorPolling();
    log('System ready', 'info');
});

// Navigation
function initNavigation() {
    document.querySelectorAll('.nav-btn').forEach(function(btn) {
        btn.addEventListener('click', function() {
            var portal = this.getAttribute('data-portal');
            switchPortal(portal);
        });
    });
}

function switchPortal(portalName) {
    document.querySelectorAll('.nav-btn').forEach(function(btn) {
        btn.classList.remove('active');
        if (btn.getAttribute('data-portal') === portalName) {
            btn.classList.add('active');
        }
    });
    
    document.querySelectorAll('.portal').forEach(function(portal) {
        portal.classList.remove('active');
    });
    document.getElementById(portalName + '-portal').classList.add('active');
    
    if (portalName === 'monitor') {
        refreshMonitoringData();
        startMonitorPolling();
    } else {
        stopMonitorPolling();
    }
    if (portalName === 'driver') {
        refreshDriversList();
        startDriverPolling();
    } else {
        stopDriverPolling();
    }
}

// =====================
// USER PORTAL
// =====================

function initUserPortal() {
    document.getElementById('registerUserBtn').addEventListener('click', registerUser);
    document.getElementById('requestRideBtn').addEventListener('click', requestRide);
    document.getElementById('cancelRideBtn').addEventListener('click', cancelRide);
    document.getElementById('newRideBtn').addEventListener('click', resetUserPortal);
    
    // Ride type selection
    document.querySelectorAll('.ride-type input').forEach(function(input) {
        input.addEventListener('change', function() {
            document.querySelectorAll('.ride-type').forEach(function(rt) {
                rt.classList.remove('selected');
            });
            this.closest('.ride-type').classList.add('selected');
        });
    });

    // Payment method selection
    document.querySelectorAll('.payment-method input').forEach(function(input) {
        input.addEventListener('change', function() {
            document.querySelectorAll('.payment-method').forEach(function(pm) {
                pm.classList.remove('selected');
            });
            this.closest('.payment-method').classList.add('selected');
        });
    });
}

function registerUser() {
    var name = document.getElementById('userName').value.trim();
    var phone = document.getElementById('userPhone').value.trim();
    
    if (!name || !phone) {
        showStatus('userStatus', 'Please fill all fields', 'error');
        return;
    }
    
    fetch(API_BASE + '/v1/users/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name, phone: phone })
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.id) {
            state.currentUser = data;
            showStatus('userStatus', 'Welcome, ' + data.name + '!', 'success');
            showLoggedInUser(data);
            log('User registered: ' + data.name, 'success');
        } else if (data.user) {
            state.currentUser = data.user;
            showStatus('userStatus', 'Welcome back, ' + data.user.name + '!', 'success');
            showLoggedInUser(data.user);
            log('User logged in: ' + data.user.name, 'info');
        } else {
            showStatus('userStatus', data.error || 'Registration failed', 'error');
        }
    })
    .catch(function(err) {
        showStatus('userStatus', 'Error: ' + err.message, 'error');
    });
}

function showLoggedInUser(user) {
    var el = document.getElementById('loggedInUser');
    el.innerHTML = '✅ Logged in as: <strong>' + user.name + '</strong> (' + user.phone + ')';
    el.style.display = 'block';
}

function requestRide() {
    if (!state.currentUser) {
        showStatus('userStatus', 'Please register first!', 'error');
        return;
    }
    
    var pickupLat = parseFloat(document.getElementById('pickupLat').value);
    var pickupLng = parseFloat(document.getElementById('pickupLng').value);
    var destLat = parseFloat(document.getElementById('destLat').value);
    var destLng = parseFloat(document.getElementById('destLng').value);
    var tier = document.querySelector('input[name="rideType"]:checked').value;
    var paymentMethod = document.querySelector('input[name="paymentMethod"]:checked').value;
    
    document.getElementById('requestRideBtn').disabled = true;
    document.getElementById('requestRideBtn').textContent = 'Searching...';
    
    fetch(API_BASE + '/v1/rides', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Idempotency-Key': 'ride-' + Date.now()
        },
        body: JSON.stringify({
            rider_id: state.currentUser.id,
            pickup_lat: pickupLat,
            pickup_lng: pickupLng,
            destination_lat: destLat,
            destination_lng: destLng,
            tier: tier,
            payment_method: paymentMethod
        })
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.id) {
            state.currentRideId = data.id;
            state.currentTripId = null;
            showRideTracking(data);
            startUserPolling();
            log('Ride created: ' + data.id.substring(0, 8), 'success');
            
            if (data.driver_assigned) {
                log('Driver assigned: ' + data.assigned_driver_id.substring(0, 8), 'success');
            } else {
                log('Waiting for driver...', 'info');
            }
        } else {
            showStatus('userStatus', data.error || 'Failed to create ride', 'error');
        }
    })
    .catch(function(err) {
        showStatus('userStatus', 'Error: ' + err.message, 'error');
    })
    .finally(function() {
        document.getElementById('requestRideBtn').disabled = false;
        document.getElementById('requestRideBtn').textContent = '🚖 Request Ride';
    });
}

function showRideTracking(ride) {
    document.getElementById('rideRequestCard').style.display = 'none';
    document.getElementById('rideTrackingCard').style.display = 'block';
    document.getElementById('cancelRideBtn').style.display = 'inline-block';
    resetTrackerSteps();
    updateRideUI(ride);
}

function resetTrackerSteps() {
    ['step-assigned', 'step-started', 'step-ended'].forEach(function(id) {
        var el = document.getElementById(id);
        if (el) el.classList.remove('completed');
    });
    ['line1', 'line2', 'line3'].forEach(function(id) {
        var el = document.getElementById(id);
        if (el) el.classList.remove('completed');
    });
    document.getElementById('fareRow').style.display = 'none';
    document.getElementById('tripCompletedSection').style.display = 'none';
    document.getElementById('newRideBtn').style.display = 'none';
    document.getElementById('cancelRideBtn').style.display = 'inline-block';
}

function updateRideUI(ride) {
    document.getElementById('userRideId').textContent = ride.id.substring(0, 8) + '...';
    
    var statusEl = document.getElementById('userRideStatus');
    statusEl.textContent = ride.status;
    statusEl.className = 'status-badge status-' + ride.status;
    
    var driverEl = document.getElementById('userDriverId');
    if (ride.assigned_driver_id) {
        driverEl.textContent = ride.assigned_driver_id.substring(0, 8) + '...';
        driverEl.style.color = '#10b981';
        document.getElementById('step-assigned').classList.add('completed');
        document.getElementById('line1').classList.add('completed');
    } else {
        driverEl.textContent = 'Searching for drivers...';
        driverEl.style.color = '#f59e0b';
    }

    // Handle different ride statuses
    if (ride.status === 'IN_TRIP') {
        statusEl.textContent = 'IN PROGRESS';
        document.getElementById('step-started').classList.add('completed');
        document.getElementById('line2').classList.add('completed');
        document.getElementById('cancelRideBtn').style.display = 'none';
    }

    if (ride.status === 'COMPLETED') {
        statusEl.textContent = 'COMPLETED';
        document.getElementById('step-started').classList.add('completed');
        document.getElementById('line2').classList.add('completed');
        document.getElementById('step-ended').classList.add('completed');
        document.getElementById('line3').classList.add('completed');
        document.getElementById('tripCompletedSection').style.display = 'block';
        document.getElementById('cancelRideBtn').style.display = 'none';
        document.getElementById('newRideBtn').style.display = 'block';
        stopUserPolling();
    }
    
    // Update surge display
    var surgeEl = document.getElementById('userSurge');
    if (surgeEl) {
        var surgeMultiplier = ride.surge_multiplier || 1;
        surgeEl.textContent = surgeMultiplier.toFixed(1) + 'x';
        if (ride.surge_active || surgeMultiplier > 1) {
            surgeEl.className = 'value surge-badge surge-active';
            surgeEl.textContent = '🔥 ' + surgeMultiplier.toFixed(1) + 'x Surge';
        } else {
            surgeEl.className = 'value surge-badge';
            surgeEl.textContent = '✓ No Surge';
        }
    }

    // Update payment method
    var paymentEl = document.getElementById('userPaymentMethod');
    if (paymentEl && ride.payment_method) {
        var paymentIcons = { 'CASH': '💵', 'CARD': '💳', 'UPI': '📱', 'WALLET': '👛' };
        paymentEl.textContent = (paymentIcons[ride.payment_method] || '') + ' ' + ride.payment_method;
    }

    // Hide cancel button if ride is cancelled
    if (ride.status === 'CANCELLED') {
        document.getElementById('cancelRideBtn').style.display = 'none';
        document.getElementById('newRideBtn').style.display = 'block';
        statusEl.textContent = 'CANCELLED';
        stopUserPolling();
    }
}

function cancelRide() {
    if (!state.currentRideId) return;
    
    if (!confirm('Are you sure you want to cancel this ride?')) return;
    
    fetch(API_BASE + '/v1/rides/' + state.currentRideId + '/cancel', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            cancelled_by: state.currentUser.id,
            reason: 'Cancelled by user'
        })
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.status === 'CANCELLED') {
            log('Ride cancelled', 'info');
            updateRideUI(data);
        } else {
            log('Failed to cancel: ' + (data.error || 'Unknown error'), 'error');
        }
    })
    .catch(function(err) {
        log('Error cancelling: ' + err.message, 'error');
    });
}

function resetUserPortal() {
    stopUserPolling();
    state.currentRideId = null;
    state.currentTripId = null;
    document.getElementById('rideTrackingCard').style.display = 'none';
    document.getElementById('rideRequestCard').style.display = 'block';
    resetTrackerSteps();
    log('Ready for new ride', 'info');
}

function startUserPolling() {
    stopUserPolling();
    state.pollingTimer = setInterval(pollRideStatus, POLL_INTERVAL);
}

function stopUserPolling() {
    if (state.pollingTimer) {
        clearInterval(state.pollingTimer);
        state.pollingTimer = null;
    }
}

function pollRideStatus() {
    if (!state.currentRideId) return;
    
    fetch(API_BASE + '/v1/rides/' + state.currentRideId)
    .then(function(r) { return r.json(); })
    .then(function(ride) {
        updateRideUI(ride);
    })
    .catch(function(err) {
        console.error('Poll error:', err);
    });
}

// =====================
// DRIVER PORTAL
// =====================

function initDriverPortal() {
    document.getElementById('registerDriverBtn').addEventListener('click', registerDriver);
    document.getElementById('refreshDriversBtn').addEventListener('click', refreshDriversList);
}

function registerDriver() {
    var name = document.getElementById('driverName').value.trim();
    var phone = document.getElementById('driverPhone').value.trim();
    var tier = document.getElementById('driverTier').value;
    
    if (!name || !phone) {
        showStatus('driverStatusMsg', 'Please fill all fields', 'error');
        return;
    }
    
    fetch(API_BASE + '/v1/drivers/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name, phone: phone, tier: tier })
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.id) {
            showStatus('driverStatusMsg', 'Driver registered: ' + data.name, 'success');
            log('Driver registered: ' + data.name, 'success');
            document.getElementById('driverName').value = '';
            document.getElementById('driverPhone').value = '';
            refreshDriversList();
        } else if (data.driver) {
            showStatus('driverStatusMsg', 'Welcome back, ' + data.driver.name + '!', 'success');
            refreshDriversList();
        } else {
            showStatus('driverStatusMsg', data.error || 'Registration failed', 'error');
        }
    })
    .catch(function(err) {
        showStatus('driverStatusMsg', 'Error: ' + err.message, 'error');
    });
}

function refreshDriversList() {
    // Fetch drivers and rides
    Promise.all([
        fetch(API_BASE + '/v1/drivers').then(function(r) { return r.json(); }),
        fetch(API_BASE + '/v1/rides').then(function(r) { return r.json(); })
    ])
    .then(function(results) {
        var drivers = results[0] || [];
        var rides = results[1] || [];
        
        renderDriverCards(drivers, rides);
    })
    .catch(function(err) {
        console.error('Failed to refresh drivers:', err);
    });
}

function renderDriverCards(drivers, rides) {
    var container = document.getElementById('driversGrid');
    
    if (!drivers || drivers.length === 0) {
        container.innerHTML = '<p class="empty-state">No drivers registered yet. Register above!</p>';
        return;
    }
    
    var html = '';
    drivers.forEach(function(driver) {
        var isOnline = driver.status === 'ONLINE';
        var isOnTrip = driver.status === 'ON_TRIP';
        
        // Find ASSIGNED rides for this driver (waiting for acceptance)
        var assignedRides = rides.filter(function(r) {
            return r.assigned_driver_id === driver.id && r.status === 'ASSIGNED';
        });

        // Find IN_TRIP rides for this driver (active trip)
        var inTripRides = rides.filter(function(r) {
            return r.assigned_driver_id === driver.id && r.status === 'IN_TRIP';
        });
        
        var cardClass = 'driver-card';
        if (assignedRides.length > 0) cardClass += ' has-ride';
        if (isOnTrip || inTripRides.length > 0) cardClass += ' on-trip';
        
        html += '<div class="' + cardClass + '" data-driver-id="' + driver.id + '">';
        html += '<div class="driver-card-header">';
        html += '<div class="driver-info">';
        html += '<span class="driver-name">' + (driver.name || 'Driver') + '</span>';
        html += '<span class="driver-tier tier-' + driver.tier + '">' + driver.tier + '</span>';
        html += '</div>';
        html += '<div class="driver-status-toggle">';
        
        if (isOnTrip || inTripRides.length > 0) {
            html += '<span class="status-indicator on-trip">🚗 ON TRIP</span>';
        } else if (isOnline) {
            html += '<span class="status-indicator online">🟢 ONLINE</span>';
        } else {
            html += '<button class="btn-small btn-success" onclick="setDriverOnline(\'' + driver.id + '\')">Go Online</button>';
        }
        
        html += '</div>';
        html += '</div>';
        
        html += '<div class="driver-id">ID: ' + driver.id.substring(0, 8) + '...</div>';
        
        // Show ASSIGNED rides that need acceptance (only when driver is ONLINE, not ON_TRIP)
        if (assignedRides.length > 0 && isOnline && !isOnTrip) {
            assignedRides.forEach(function(ride) {
                html += '<div class="ride-alert">';
                html += '<div class="ride-alert-header">🔔 New Ride Request!</div>';
                html += '<div class="ride-details">';
                html += '<div><strong>Ride:</strong> ' + ride.id.substring(0, 8) + '...</div>';
                html += '<div><strong>Pickup:</strong> ' + ride.pickup_lat.toFixed(4) + ', ' + ride.pickup_lng.toFixed(4) + '</div>';
                html += '<div><strong>Destination:</strong> ' + ride.destination_lat.toFixed(4) + ', ' + ride.destination_lng.toFixed(4) + '</div>';
                if (ride.surge_active) {
                    html += '<div class="surge-info">🔥 Surge: ' + (ride.surge_multiplier || 1).toFixed(1) + 'x</div>';
                }
                html += '</div>';
                html += '<div class="ride-actions">';
                html += '<button class="btn-success btn-large" onclick="acceptRide(\'' + driver.id + '\', \'' + ride.id + '\')">✅ Accept & Start Trip</button>';
                html += '</div>';
                html += '</div>';
            });
        }
        
        // Show IN_TRIP rides with trip controls
        if (inTripRides.length > 0) {
            var ride = inTripRides[0]; // Should only be one
            var tripInfo = state.driverTrips[driver.id];
            
            html += '<div class="trip-active">';
            html += '<div class="trip-header">🚗 Trip in Progress</div>';
            html += '<div class="trip-details">';
            html += '<div>Ride: ' + ride.id.substring(0, 8) + '...</div>';
            if (tripInfo && tripInfo.status === 'PAUSED') {
                html += '<div class="trip-paused-notice">⏸️ Trip is PAUSED</div>';
            }
            html += '</div>';
            html += '<div class="trip-actions">';
            
            if (tripInfo) {
                if (tripInfo.status === 'STARTED') {
                    html += '<button class="btn-warning" onclick="pauseTrip(\'' + driver.id + '\', \'' + tripInfo.trip_id + '\')">⏸️ Pause</button>';
                } else if (tripInfo.status === 'PAUSED') {
                    html += '<button class="btn-success" onclick="resumeTrip(\'' + driver.id + '\', \'' + tripInfo.trip_id + '\')">▶️ Resume</button>';
                }
                html += '<button class="btn-danger" onclick="endTrip(\'' + driver.id + '\', \'' + tripInfo.trip_id + '\')">🏁 End Trip</button>';
            }
            
            html += '</div>';
            html += '</div>';
        }
        
        html += '</div>';
    });
    
    container.innerHTML = html;
}

function setDriverOnline(driverId) {
    // Generate random location near Bangalore
    var lat = 12.9716 + (Math.random() - 0.5) * 0.02;
    var lng = 77.5946 + (Math.random() - 0.5) * 0.02;
    
    fetch(API_BASE + '/v1/drivers/' + driverId + '/location', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ lat: lat, lng: lng })
    })
    .then(function(r) {
        if (r.ok) {
            log('Driver ' + driverId.substring(0, 8) + ' is now ONLINE', 'success');
            refreshDriversList();
        }
    })
    .catch(function(err) {
        log('Failed to go online: ' + err.message, 'error');
    });
}

function acceptRide(driverId, rideId) {
    log('Accepting ride ' + rideId.substring(0, 8) + '...', 'info');
    
    fetch(API_BASE + '/v1/drivers/' + driverId + '/accept', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ride_id: rideId })
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.trip_id) {
            // Store trip info for this driver
            state.driverTrips[driverId] = {
                trip_id: data.trip_id,
                ride_id: rideId,
                status: 'STARTED'
            };
            
            log('Trip started: ' + data.trip_id.substring(0, 8), 'success');
            
            // Update user portal if this is their ride
            if (state.currentRideId === rideId) {
                state.currentTripId = data.trip_id;
            }
            
            refreshDriversList();
        } else {
            log('Failed to accept: ' + (data.error || 'Unknown error'), 'error');
        }
    })
    .catch(function(err) {
        log('Error accepting ride: ' + err.message, 'error');
    });
}

function pauseTrip(driverId, tripId) {
    log('Pausing trip ' + tripId.substring(0, 8) + '...', 'info');
    
    fetch(API_BASE + '/v1/trips/' + tripId + '/pause', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.status === 'PAUSED') {
            if (state.driverTrips[driverId]) {
                state.driverTrips[driverId].status = 'PAUSED';
            }
            log('Trip paused', 'info');
            refreshDriversList();
        } else {
            log('Failed to pause: ' + (data.error || 'Unknown error'), 'error');
        }
    })
    .catch(function(err) {
        log('Error pausing trip: ' + err.message, 'error');
    });
}

function resumeTrip(driverId, tripId) {
    log('Resuming trip ' + tripId.substring(0, 8) + '...', 'info');
    
    fetch(API_BASE + '/v1/trips/' + tripId + '/resume', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.status === 'STARTED') {
            if (state.driverTrips[driverId]) {
                state.driverTrips[driverId].status = 'STARTED';
            }
            log('Trip resumed', 'success');
            refreshDriversList();
        } else {
            log('Failed to resume: ' + (data.error || 'Unknown error'), 'error');
        }
    })
    .catch(function(err) {
        log('Error resuming trip: ' + err.message, 'error');
    });
}

function endTrip(driverId, tripId) {
    log('Ending trip ' + tripId.substring(0, 8) + '...', 'info');
    
    fetch(API_BASE + '/v1/trips/' + tripId + '/end', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.fare !== undefined) {
            // Clear trip info for this driver
            delete state.driverTrips[driverId];
            
            log('🎉 Trip completed! Earned: ₹' + data.fare.toFixed(2), 'success');
            
            // Show earnings notification
            showEarningsNotification(driverId, data.fare);
            
            // Refresh the driver list
            refreshDriversList();
        } else {
            log('Failed to end trip: ' + (data.error || 'Unknown error'), 'error');
        }
    })
    .catch(function(err) {
        log('Error ending trip: ' + err.message, 'error');
    });
}

function showEarningsNotification(driverId, fare) {
    // Create a temporary earnings notification
    var notification = document.createElement('div');
    notification.className = 'earnings-notification';
    notification.innerHTML = '💰 Trip completed! You earned <strong>₹' + fare.toFixed(2) + '</strong>';
    
    var driverCard = document.querySelector('[data-driver-id="' + driverId + '"]');
    if (driverCard) {
        driverCard.appendChild(notification);
        
        // Remove after 5 seconds
        setTimeout(function() {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);
    }
}

function startDriverPolling() {
    stopDriverPolling();
    state.driverPollingTimer = setInterval(refreshDriversList, 3000);
    
    // Animate live indicator
    var indicator = document.getElementById('liveIndicator');
    if (indicator) indicator.classList.add('active');
}

function stopDriverPolling() {
    if (state.driverPollingTimer) {
        clearInterval(state.driverPollingTimer);
        state.driverPollingTimer = null;
    }
    
    var indicator = document.getElementById('liveIndicator');
    if (indicator) indicator.classList.remove('active');
}

// =====================
// MONITORING PORTAL
// =====================

function initMonitoringPortal() {
    document.getElementById('refreshDataBtn').addEventListener('click', refreshMonitoringData);
}

function startMonitorPolling() {
    if (state.monitorPollingTimer) {
        clearInterval(state.monitorPollingTimer);
    }
    state.monitorPollingTimer = setInterval(refreshMonitoringData, 3000); // Refresh every 3 seconds
}

function stopMonitorPolling() {
    if (state.monitorPollingTimer) {
        clearInterval(state.monitorPollingTimer);
        state.monitorPollingTimer = null;
    }
}

function refreshMonitoringData() {
    try {
        fetchUsers();
        fetchDriversTable();
        fetchRides();
    } catch (e) {
        console.error('Error refreshing:', e);
    }
}

function fetchUsers() {
    fetch(API_BASE + '/v1/users')
    .then(function(r) { return r.json(); })
    .then(function(users) {
        var tbody = document.getElementById('usersTableBody');
        if (!users || users.length === 0) {
            tbody.innerHTML = '<tr><td colspan="3" class="empty">No users registered</td></tr>';
            return;
        }
        
        var html = '';
        users.forEach(function(u) {
            html += '<tr>';
            html += '<td>' + u.id.substring(0, 8) + '...</td>';
            html += '<td>' + u.name + '</td>';
            html += '<td>' + u.phone + '</td>';
            html += '</tr>';
        });
        tbody.innerHTML = html;
    })
    .catch(function(err) { console.error('fetchUsers error:', err); });
}

function fetchDriversTable() {
    // Fetch drivers and trips to calculate real earnings
    Promise.all([
        fetch(API_BASE + '/v1/drivers').then(function(r) { return r.json(); }),
        fetch(API_BASE + '/v1/trips').then(function(r) { return r.json(); })
    ])
    .then(function(results) {
        var drivers = results[0] || [];
        var trips = results[1] || [];
        
        var tbody = document.getElementById('driversTableBody');
        if (!drivers || drivers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="empty">No drivers</td></tr>';
            document.getElementById('onlineDrivers').textContent = '0';
            return;
        }

        // Calculate earnings per driver from ENDED trips (has actual fare)
        var driverEarnings = {};
        trips.forEach(function(t) {
            if (t.status === 'ENDED' && t.driver_id) {
                if (!driverEarnings[t.driver_id]) {
                    driverEarnings[t.driver_id] = { total: 0, trips: 0 };
                }
                driverEarnings[t.driver_id].total += t.fare || 0;
                driverEarnings[t.driver_id].trips += 1;
            }
        });

        var html = '';
        var online = 0;
        drivers.forEach(function(d) {
            if (d.status === 'ONLINE' || d.status === 'ON_TRIP') online++;
            var earnings = driverEarnings[d.id] || { total: 0, trips: 0 };

            html += '<tr>';
            html += '<td>' + d.id.substring(0, 8) + '...</td>';
            html += '<td>' + (d.name || '-') + '</td>';
            html += '<td><span class="status-badge status-' + d.status + '">' + d.status + '</span></td>';
            html += '<td>' + d.tier + '</td>';
            html += '<td class="earnings">₹' + earnings.total.toFixed(2) + ' <span class="trips-count">(' + earnings.trips + ' trips)</span></td>';
            html += '</tr>';
        });
        tbody.innerHTML = html;
        document.getElementById('onlineDrivers').textContent = online;
    })
    .catch(function(err) { console.error('fetchDriversTable error:', err); });
}

function fetchRides() {
    fetch(API_BASE + '/v1/rides')
    .then(function(r) { return r.json(); })
    .then(function(rides) {
        var tbody = document.getElementById('ridesTableBody');
        if (!rides || rides.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="empty">No rides</td></tr>';
            document.getElementById('totalRides').textContent = '0';
            document.getElementById('activeTrips').textContent = '0';
            return;
        }
        
        var html = '';
        var active = 0;
        rides.forEach(function(r) {
            // Count active rides (REQUESTED, ASSIGNED, IN_TRIP)
            if (r.status === 'REQUESTED' || r.status === 'ASSIGNED' || r.status === 'IN_TRIP') {
                active++;
            }
            
            html += '<tr>';
            html += '<td>' + r.id.substring(0, 8) + '...</td>';
            html += '<td>' + (r.rider_id || '-').substring(0, 8) + '...</td>';
            html += '<td>' + (r.assigned_driver_id ? r.assigned_driver_id.substring(0, 8) + '...' : '-') + '</td>';
            html += '<td><span class="status-badge status-' + r.status + '">' + r.status + '</span></td>';
            html += '<td>' + (r.payment_method || '-') + '</td>';
            html += '</tr>';
        });
        tbody.innerHTML = html;
        document.getElementById('totalRides').textContent = rides.length;
        document.getElementById('activeTrips').textContent = active;
    })
    .catch(function(err) { console.error('fetchRides error:', err); });
}

// =====================
// UTILITIES
// =====================

function showStatus(elementId, message, type) {
    var el = document.getElementById(elementId);
    el.textContent = message;
    el.className = 'status-msg ' + type;
    el.style.display = 'block';
    
    setTimeout(function() {
        el.style.display = 'none';
    }, 5000);
}

function log(message, type) {
    var logContainer = document.getElementById('activityLog');
    if (!logContainer) return;
    
    var entry = document.createElement('div');
    entry.className = 'log-entry ' + (type || '');
    entry.textContent = '[' + new Date().toLocaleTimeString() + '] ' + message;
    logContainer.insertBefore(entry, logContainer.firstChild);
    
    while (logContainer.children.length > 50) {
        logContainer.removeChild(logContainer.lastChild);
    }
}
