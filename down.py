import gdown

# ID file = phần sau '/d/' và trước '/edit'
file_id = "1QH61dZeUM9QxQnaFA5Wo314KXmdAIzLh"
url = f"https://drive.google.com/uc?id={file_id}"

# Tải file
gdown.download(url, output="output.pdf", quiet=False)
