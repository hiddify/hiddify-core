Pod::Spec.new do |s|    
    s.name             = 'Libcore'
    s.version          = '0.10.0'
    s.summary          = 'hiddify mobile SDK for iOS'  
    s.homepage         = 'https://hiddify.com/'
    s.license          = { :type => 'Copyright', :text => 'Hiddify Open Software' }
    s.author           = { 'Hiddify' => 'ios@hiddify.com' }
    
    s.ios.deployment_target = '9.0'
    s.vendored_frameworks = 'Libcore.xcframework'
    # s.source = { :git => 'https://github.com/hiddify/hiddify-next-core.git', :tag => s.version }
    s.source           = { :http => "https://github.com/hiddify/hiddify-next-core/releases/download/v#{s.version}/hiddify-libcore-ios.xcframework.tar.gz" }
    # s.prepare_command = <<-CMD
    #   ls -R -l
    #   tar -xf "${PODS_TARGET_SRCROOT}/hiddify-libcore-ios.xcframework.tar.gz"
    # CMD



  end
  
