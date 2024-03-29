document.addEventListener("DOMContentLoaded", function () {
    // 使用XMLHttpRequest来进行POST请求
    var xhr = new XMLHttpRequest();
    xhr.open("GET", BASE_URL + "/get_toMe", true);
    xhr.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
  
    // 当接收到响应时的操作
    xhr.onload = function () {
      if (xhr.status === 200) {
        var response = JSON.parse(xhr.responseText);
        populateTable(response.applications);
      } else {
        console.error("Server error:", xhr.status);
      }
    };
  
    xhr.onerror = function () {
      console.error("Request failed", xhr.status);
    };
  
    xhr.send();
  });

function addApplicationRow(application, tableBody) {
    const row = tableBody.insertRow();
    console.log("addApplicationRow:",application);
  
    const rows = [
      "ServiceName",
      "ServiceID",
      "InitiatorID",
      "ApplicationTime",
      "Status",
      "Operation"
    ];
  
    row.insertCell(rows.indexOf("ServiceName")).textContent = application.ServiceName;
    row.insertCell(rows.indexOf("ServiceID")).textContent = application.ServiceID;
    row.insertCell(rows.indexOf("InitiatorID")).textContent = application.InitiatorID;
    row.insertCell(rows.indexOf("ApplicationTime")).textContent = application.ApplicationTime
    row.insertCell(rows.indexOf("Status")).textContent = application.Status == 0 ? "Pending" : application.Status == 1 ? "Approved" : "Rejected";

    // 添加approve按钮
    const approveBtn = document.createElement("button");
    approveBtn.className = "button-style";
    approveBtn.textContent = "Approve";
    // approveBtn.setAttribute("data-service-id", application.ServiceID);
    approveBtn.onclick = function () {
        const answer = {
            InitiatorID : application.InitiatorID,
            ServiceID : application.ServiceID,
            Status : 1,
            ApplicationTime : application.ApplicationTime,
            ProcessTime : new Date().toLocaleString()
        }
        fetch("/approve_application", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(answer),
        })
        .then((response) => {
            if (response.ok) {
                console.log("Application approved successfully");
                // 刷新页面
                location.reload();
            } else {
                console.error("Failed to approve application");
            }
        })
    };

    const rejectBtn = document.createElement("button");
    rejectBtn.className = "button-style";
    rejectBtn.textContent = "Reject";
    // approveBtn.setAttribute("data-service-id", application.ServiceID);
    rejectBtn.onclick = function () {
        const answer = {
            InitiatorID : application.InitiatorID,
            ServiceID : application.ServiceID,
            Status : 2,
            ApplicationTime : application.ApplicationTime,
            ProcessTime : new Date().toLocaleString()
        }
        fetch("/approve_application", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(answer),
        })
        .then((response) => {
            if (response.ok) {
                console.log("Application rejected successfully");
                // 刷新页面
                location.reload();
            } else {
                console.error("Failed to reject application");
            }
        })
    };
    const operationCell = row.insertCell(rows.indexOf("Operation"));
    operationCell.className = "operation";
    const buttonContainer = document.createElement("div");
    buttonContainer.className = "button-container";
    buttonContainer.appendChild(approveBtn);
    buttonContainer.appendChild(rejectBtn);
    operationCell.appendChild(buttonContainer);

    // disable处理过的btn
    if (application.Status != 0) {
        approveBtn.disabled = true;
        approveBtn.classList.remove("button-style");
        approveBtn.classList.add("button-disabled");

        rejectBtn.disabled = true;
        rejectBtn.classList.remove("button-style");
        rejectBtn.classList.add("button-disabled");
    }

}

function populateTable(applications) {
    const tableBody = document
      .getElementById("applicationTable")
      .getElementsByTagName("tbody")[0];
  
    applications.forEach((application) => {
      addApplicationRow(application, tableBody);
    });
  }