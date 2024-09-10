
read -p "Do you want to clean cert? (y/n): " choice
case "$choice" in
  y|Y|yes|YES)
    echo "Proceeding with the operation..."
    sudo rm -rf ~/.m-cmp/data/certbot
    ;;
  n|N|no|NO)
    echo "Operation cancelled."
    exit 0
    ;;
  *)
    echo "Invalid input. Please enter 'yes' or 'no'."
    exit 1
    ;;
esac