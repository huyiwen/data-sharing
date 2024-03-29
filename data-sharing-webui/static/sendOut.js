document.addEventListener("DOMContentLoaded", function () {
  // 使用XMLHttpRequest来进行POST请求
  var xhr = new XMLHttpRequest();
  console.log("BASE_URL:",BASE_URL)
  xhr.open("GET", BASE_URL + "/get_sendOut", true);
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
    "ProcessTime",
    "Status",
    "Operation"
  ];

  row.insertCell(rows.indexOf("ServiceName")).textContent = application.ServiceName;
  row.insertCell(rows.indexOf("ServiceID")).textContent = application.ServiceID;
  row.insertCell(rows.indexOf("InitiatorID")).textContent = application.InitiatorID;
  row.insertCell(rows.indexOf("ApplicationTime")).textContent = application.ApplicationTime
  row.insertCell(rows.indexOf("ProcessTime")).textContent = application.Status ? application.ProcessTime : "N/A";
  row.insertCell(rows.indexOf("Status")).textContent = application.Status == 0 ? "Pending" : application.Status == 1 ? "Approved" : "Rejected";

  // 为弹窗添加关闭按钮
  // 在弹窗中找到关闭按钮
  // const closeButton = document.querySelector("#popup .close");
  // // 为关闭按钮添加点击事件处理程序
  // closeButton.addEventListener("click", function () {
  //   // 隐藏弹窗
  //   document.getElementById("popup").style.display = "none";
  // });

  // 添加approve按钮
  const viewBtn = document.createElement("button");
  viewBtn.className = "button-style";
  viewBtn.textContent = "ViewData";
  // approveBtn.setAttribute("data-service-id", application.ServiceID);
  viewBtn.onclick = function () {
      // fetch_data 
      const data = {
        ServiceID : application.ServiceID,
        PublisherURL: application.PublisherURL
      }
      fetch(BASE_URL + "/fetch_data", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(data),
      })
      .then((response) => {
        console.log(response)
        if (!response.ok) {
          throw new Error("Network response was not ok " + response.statusText);
        }
        return response.json();
      })
      .then((data) => {
        displayDataModal(data);
      })
      .catch((error) => {
        console.error("Error fetching data:", error);
      });
      // .then((response) => response.json())
      // .then((data) => {
      //   const table = createTable(data);
      //   const popup = document.getElementById("popup");
      //   popup.innerHTML = "";
      //   popup.appendChild(closeButton); // 将关闭按钮添加到弹窗中
      //   popup.appendChild(table);
      //   popup.style.display = "block";
      // });
  };
  row.insertCell(rows.indexOf("Operation")).appendChild(viewBtn);
}

function populateTable(applications) {
  const tableBody = document
    .getElementById("applicationTable")
    .getElementsByTagName("tbody")[0];

  applications.forEach((application) => {
    addApplicationRow(application, tableBody);
  });
}

function createTable(data) {
  const table = document.createElement("table");
  table.className = "data-table"; // Apply CSS styling as needed
  const thead = document.createElement("thead");
  const tbody = document.createElement("tbody");

  console.log("datatype:", typeof(data))

  // 创建表头
  const headerRow = document.createElement("tr");
  for (const key in data[0]) {
    const th = document.createElement("th");
    th.textContent = key;
    headerRow.appendChild(th);
  }
  thead.appendChild(headerRow);

  // 创建表格内容
  for (const item of data) {
    const row = document.createElement("tr");
    for (const key in item) {
      const td = document.createElement("td");
      td.textContent = item[key];
      row.appendChild(td);
    }
    tbody.appendChild(row);
  }

  table.appendChild(thead);
  table.appendChild(tbody);

  return table;
}

function displayDataModal(responseData) {
  const modal = document.createElement("div");
  modal.className = "modal";

  modal.onclick = function (event) {
    if (event.target === modal) {
      document.body.removeChild(modal); // 从DOM中移除整个遮罩层和模态窗口
    }
  };

  const modalContent = document.createElement("div");
  modalContent.className = "modal-content";
  modal.appendChild(modalContent);

  const closeBtn = document.createElement("span");
  closeBtn.className = "close";
  closeBtn.innerHTML = "&times;";
  closeBtn.onclick = function () {
    modal.style.display = "none";
    document.body.removeChild(modal);
  };
  modalContent.appendChild(closeBtn);

  console.log("data:", (responseData))
 
  const resdata = JSON.parse(responseData.data)
  table = createTable(resdata);

  modalContent.appendChild(table);
  document.body.appendChild(modal);
  modal.style.display = "block";
}