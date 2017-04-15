require 'etc'

$init = <<SCRIPT
sudo apt install git valgrind gcc clang make \
  pkg-config manpages-dev build-essential gdb \
  libexpat-dev libsqlite3-dev libxml2-dev \
  libxml2-utils libmagic-dev

mkdir -p /home/vagrant/go/src/github.com/shabbyrobe
ln -s /vagrant /home/vagrant/go/src/github.com/shabbyrobe/xmlwriter
SCRIPT

Vagrant.configure(2) do |config|
  config.vm.box_check_update = false

  config.vm.provision "shell", inline: $init

  # config.vm.synced_folder "../", "/srv/vidiot"

  config.vm.box = "boxcutter/ubuntu1604"

  if Vagrant.has_plugin?("vagrant-cachier")
    config.cache.scope = :box
    config.cache.enable :apt
  end

  config.vm.provider "virtualbox" do |vb|
    vb.memory = 1024
    vb.cpus = Etc.nprocessors
  end
end
