$(function() {
    $("input:radio[name=protocol]")[0].checked=true;
    $("input:radio[name=protocol]").change();
    $("#mandate").prop("checked", true);
    $("#mandate").change();
    refreshUserData();
});

$("input:radio[name=protocol]").change(function () {
    let port = $("input[name=protocol]:checked").data("default-port");
    if ($("#port").val() < 1024) {
        $("#port").val(port);
    }
});

$("#mandate").change(function() {
    $(".accessParameters").hide();
    $("#mandateAccessParameters").show();
});

$("#custom").change(function() {
    $(".accessParameters").hide();
    $("#customAccessParameters").show();
});

$("#createSessionButton").click(function () {
    $.post("/api/sessions", {
        hostname:          $("#hostname").val(),
        port:              $("#port").val(),
        protocol_id:       $("input[name=protocol]:checked").val(),
        access_type:       $("input[name=accessType]:checked").val(),
        mandate_id:        $("#mandateSelect").val(),
        custom_network_id: $("#customNetwork").val(),
        custom_login:      $("#customLogin").val(),
        custom_password:   $("#customPassword").val(),
        custom_key:        $("#customKey").val(),
    })
        .done(function( data ) {
            window.open("ssh://" + data.token + "@" + data.servicepoint,"_self")
        })
        .fail(function() {
            alert("Произошла ошибка. Сессия не создана");
        });
});

$("#saveSessionButton").click(function () {
    let hostname = $("#hostname").val();
    let port = $("#port").val();
    let access_type = $("input[name=accessType]:checked").val();
    let connectionName = $("#connectionName").val();
    if (connectionName === "") {
        connectionName = hostname+':'+port+' ('+access_type+' access)';
    }
    let st = {
        name:              connectionName,
        hostname:          hostname,
        port:              port,
        protocol_id:       $("input[name=protocol]:checked").val(),
        access_type:       access_type,
        mandate_id:        $("#mandateSelect").val(),
        custom_network_id: $("#customNetwork").val(),
        custom_login:      $("#customLogin").val(),
        custom_password:   $("#customPassword").val(),
        custom_key:        $("#customKey").val(),
    };
    $.post("/api/sessiontemplates", st)
        .done(function() {
            refreshUserData();
        })
        .fail(function() {
            alert("Произошла ошибка. Сессия не сохранена");
        });
});


var userDataCache;

function refreshUserData() {
    $.get("/api/userdata", function(data) {
        userDataCache = data;
        updateSavedConnectionsList()
    });
}

function updateSavedConnectionsList() {
    let h = $("#connParams").height();
    $(".connections").height(h);
    let stContainer = $(".connections");
    stContainer.empty();
    stContainer.append("<legend>Сохранённые подключения</legend>");
    userDataCache.session_templates.forEach(function (st, idx) {
        stContainer.append("<div class='connectionContainer'>" +
            "<a href='#' class='sessionTemplateEditButton' data-id='"+st.id+"'>"+st.name+"</a>" +
            "<span class='spacer'></span>" +
            "<button type='button' data-id='"+st.id+"' class='floatRight compactFlat sessionTemplateDeleteButton'>" +
            "    <span class='fas fa-trash-alt'></span>" +
            "</button>" +
            "</div>");
    });
    $(".sessionTemplateEditButton").click(function () {
        let sid = $(this).data("id");
        fillConnectionParametersForm(sid);
    });
    $(".sessionTemplateDeleteButton").click(function () {
        let sid = $(this).data("id");
        $.ajax({
            url: "/api/sessiontemplates/" + sid,
            type: "DELETE",
            data: { id: sid },
            success: function(result) {
                userDataCache.session_templates.forEach(function (st, idx) {
                    if (st.id === sid) {
                        userDataCache.session_templates.splice(idx, 1);
                    }
                });
                updateSavedConnectionsList()
            }
        });
    });
}

function fillConnectionParametersForm(sessionID) {
    userDataCache.session_templates.forEach(function (st, idx) {
        if (st.id === sessionID) {
            $("#connectionName").val(st.name);
            $("#hostname").val(st.target_host);
            $("#port").val(st.target_port);
            $('input:radio[name=protocol][value='+st.target_protocol_id+']').prop('checked', true);
            if (typeof st.mandate_id !== 'undefined') {
                $('input:radio[name=accessType][value=mandate]').prop('checked', true);
                $("#mandate").change();
                $('#mandateSelect').val(st.mandate_id)
            } else {
                $('input:radio[name=accessType][value=custom]').prop('checked', true);
                $("#custom").change();
            }
            $("#customNetwork").val(st.custom_target_network_id);
            $("#customLogin").val(st.custom_target_login);
            $("#customPassword").val(st.custom_target_password);
            $("#customKey").val(st.custom_target_priv_key);

        }
    });
}

$("#logoffButton").click(function () {
    alert("Разлогинивание ещё не запилили!")
});
